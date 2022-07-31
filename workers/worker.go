package workers

import (
	"bufio"
	"compare/core"
	"compare/jobs"
	"compare/log"
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Worker struct {
	id              int
	progress        chan string
	TargetBuffer    map[string]*StagingRow
	UnMatchedBuffer []*StagingRow
	Start           time.Time
	re              *regexp.Regexp
}

type StagingRow struct {
	Count int
	Raw   string
}

type CsvWriter struct {
	File   *os.File
	Writer *csv.Writer
}

type NotFoundRow struct {
	Row     string
	Type    string
	Raw     string
	Closest []*Closest
}

type Closest struct {
	Id      string
	Type    string
	Raw     string
	Indices string
}

func CreateWorker(id int, progress chan string) *Worker {
	return &Worker{
		TargetBuffer:    make(map[string]*StagingRow),
		UnMatchedBuffer: make([]*StagingRow, 0),
		progress:        progress,
		re:              regexp.MustCompile(`\r?\n`),
		Start:           time.Now(),
		id:              id,
	}
}

func (w *Worker) Execute(wg *sync.WaitGroup, jobsChan <-chan *jobs.Job, done chan<- string) {
	defer wg.Done()
	for job := range jobsChan {
		result, err := w.doComparison(job)
		if err != nil {
			log.Warn(fmt.Sprintf("failed to do comparison: %v", err))
		}
		done <- *result
	}
}

func (w *Worker) doComparison(job *jobs.Job) (*string, error) {
	log.V5(fmt.Sprintf("worker %v got job %v", w.id, job))

	err := w.createTargetBuffer(job.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to create staging buffer: %v", err)
	}

	log.V5(fmt.Sprintf("running comparison: %v - %v", job.Target, job.Source))

	result, err := w.compareBuffers(job.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to comapare buffers: %v", err)
	}

	w.findClosest(result)

	err = w.createMissingFile(result, job.SourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to write missing file: %v", err)
	}

	err = w.createRawFile(result, job.SourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to write missing file: %v", err)
	}

	err = w.createUnMatchedFile(job.SourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to write missing file: %v", err)
	}

	return &job.Source, nil
}

func (w *Worker) closeWriter(csvWriter *CsvWriter) error {
	csvWriter.Writer.Flush()
	err := csvWriter.File.Close()
	if err != nil {
		return fmt.Errorf("failed to close file: %v", err)
	}
	return nil
}

func (w *Worker) createCsvWriter(path string, header []string) (*CsvWriter, error) {
	csvFile, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed creating file %v: %s", path, err)
	}

	csvwriter := csv.NewWriter(csvFile)
	err = csvwriter.Write(header)
	if err != nil {
		return nil, fmt.Errorf("failed to write header: %v", err)
	}

	return &CsvWriter{File: csvFile, Writer: csvwriter}, nil
}

func (w *Worker) getUnmatched() {
	count := 0
	for _, target := range w.TargetBuffer {
		if target.Count > 0 {
			w.UnMatchedBuffer = append(w.UnMatchedBuffer, target)
			w.printProgress(count)
			count++
		}
	}
}

func (w *Worker) compareString(source string, target string) []string {
	result := make([]string, 0)
	if source[6:10] == target[6:10] {
		for index := range source {
			if index > 10 && source[index] != target[index] {
				result = append(result, strconv.Itoa(index))
			}
		}
	}
	return result
}

func (w *Worker) findClosest(notFoundArray []*NotFoundRow) {
	log.V5("checking for unmatched entries")
	w.getUnmatched()
	for _, notFoundRow := range notFoundArray {
		for i, unMatchedRow := range w.UnMatchedBuffer {
			if unMatchedRow.Count > 0 {
				indices := w.compareString(notFoundRow.Raw, unMatchedRow.Raw)
				if len(indices) > 0 && len(indices) <= 5 {
					closest := &Closest{
						Id:      unMatchedRow.Raw[0:6],
						Type:    unMatchedRow.Raw[6:10],
						Raw:     unMatchedRow.Raw,
						Indices: strings.Join(indices, ":"),
					}
					notFoundRow.Closest = append(notFoundRow.Closest, closest)
					w.UnMatchedBuffer[i].Count--
				}
			}
		}
	}
}

func (w *Worker) createTargetBuffer(path string) error {
	log.V5(fmt.Sprintf("Creating TargetBuffer for %v", path))
	readFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open path %v: %v", path, err)
	}

	fileScanner := bufio.NewScanner(readFile)

	fileScanner.Split(bufio.ScanLines)

	count := 0
	for fileScanner.Scan() {
		w.printProgress(count)

		text := fileScanner.Text()
		text = w.re.ReplaceAllString(text, "")
		key := w.makeKey(fileScanner.Text())

		if w.TargetBuffer[key] != nil {
			w.TargetBuffer[key].Count++
		} else {
			w.TargetBuffer[key] = &StagingRow{
				Count: 1,
				Raw:   text,
			}
		}
		count++
	}

	err = readFile.Close()
	if err != nil {
		return fmt.Errorf("failed to close path %v: %v", path, err)
	}

	return nil
}

func (w *Worker) makeKey(text string) string {
	line := text[6 : len(text)-1]
	hash := md5.Sum([]byte(line))
	return hex.EncodeToString(hash[:])
}

func (w *Worker) compareBuffers(path string) ([]*NotFoundRow, error) {
	readFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open path %v: %v", path, err)
	}

	fileScanner := bufio.NewScanner(readFile)

	fileScanner.Split(bufio.ScanLines)
	notFound := make([]*NotFoundRow, 0)

	log.V5("scanner ready")
	count := 0

	for fileScanner.Scan() {
		w.printProgress(count)
		text := fileScanner.Text()

		if text != "" {
			text = w.re.ReplaceAllString(text, "")
			key := w.makeKey(text)

			if w.TargetBuffer[key] == nil || w.TargetBuffer[key].Count <= 0 {
				item := &NotFoundRow{
					Row:  text[0:6],
					Type: text[6:10],
					Raw:  text,
				}
				notFound = append(notFound, item)
			} else {
				w.TargetBuffer[key].Count--
			}
		}

		count++
	}
	return notFound, nil
}

func (w *Worker) printProgress(count int) {
	if count%5000 == 0 {
		elapsed := time.Now().Sub(w.Start).Seconds()
		w.progress <- fmt.Sprintf("worker %v - did line %v took %v seconds", w.id, count, elapsed)
	}
}

func (w *Worker) createMissingFile(result []*NotFoundRow, source string) error {
	path := fmt.Sprintf("%v/%v-missing.csv", core.Config.ResultPath, source)
	log.V5(fmt.Sprintf("saving missing file: %v", path))

	header := []string{"source row", "source type", "target row", "changed indices"}
	csvWriter, err := w.createCsvWriter(path, header)
	if err != nil {
		return fmt.Errorf("failed to create writer: %v", err)
	}

	for _, item := range result {
		closestIdString := ""
		closestIndices := ""
		for _, closest := range item.Closest {
			closestIdString += closest.Id + "-" + closest.Type + " | "
			closestIndices += closest.Indices + "|"
		}
		row := []string{item.Row, item.Type, closestIdString, closestIndices}
		err := csvWriter.Writer.Write(row)
		if err != nil {
			return fmt.Errorf("failed to write missing file: %v", err)
		}
	}

	err = w.closeWriter(csvWriter)
	if err != nil {
		return fmt.Errorf("failed to close writer: %v", err)
	}
	return nil
}

func (w *Worker) createRawFile(result []*NotFoundRow, source string) error {
	path := fmt.Sprintf("%v/%v-raw.csv", core.Config.ResultPath, source)
	log.V5(fmt.Sprintf("saving raw file: %v", path))

	header := []string{"row", "raw"}
	csvWriter, err := w.createCsvWriter(path, header)
	if err != nil {
		return fmt.Errorf("failed to create writer: %v", err)
	}

	for _, item := range result {
		var closest *Closest
		if item.Closest == nil {
			continue
		}
		for i, closeItem := range item.Closest {
			if i == 0 {
				closest = closeItem
			}
			if len(closeItem.Indices) < len(closest.Indices) {
				closest = closeItem
			}
		}

		row := []string{item.Row, item.Raw}
		err = csvWriter.Writer.Write(row)
		if err != nil {
			return fmt.Errorf("failed to write: %v", err)
		}

		row = []string{closest.Id, closest.Raw}
		err = csvWriter.Writer.Write(row)
		if err != nil {
			return fmt.Errorf("failed to write: %v", err)
		}

		row = []string{}
		err = csvWriter.Writer.Write(row)
		if err != nil {
			return fmt.Errorf("failed to write: %v", err)
		}
	}

	err = w.closeWriter(csvWriter)
	if err != nil {
		return fmt.Errorf("failed to close writer: %v", err)
	}
	return nil
}

func (w *Worker) createUnMatchedFile(source string) error {
	path := fmt.Sprintf("%v/%v-unmatched.csv", core.Config.ResultPath, source)
	log.V5(fmt.Sprintf("saving unmatched file: %v", path))
	header := []string{"Id", "Type", "Raw"}

	csvWriter, err := w.createCsvWriter(path, header)
	if err != nil {
		return fmt.Errorf("failed to create writer: %v", err)
	}

	for _, item := range w.UnMatchedBuffer {
		if item.Count > 0 && item.Raw != "" {
			row := []string{item.Raw[0:6], item.Raw[6:10], item.Raw}
			_ = csvWriter.Writer.Write(row)
		}
	}

	err = w.closeWriter(csvWriter)
	if err != nil {
		return err
	}

	return nil
}
