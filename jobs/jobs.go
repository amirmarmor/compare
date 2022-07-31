package jobs

import (
	"compare/core"
	"fmt"
	"io/fs"
	"io/ioutil"
	"strings"
)

type Jobs struct {
	Pool        chan *Job
	SourceFiles []fs.FileInfo
	TargetFiles []fs.FileInfo
}

type Job struct {
	Target     string
	Source     string
	TargetName string
	SourceName string
}

func Create() (*Jobs, error) {
	target, err := ioutil.ReadDir(core.Config.TargetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %v: %V", core.Config.TargetPath, err)
	}

	source, err := ioutil.ReadDir(core.Config.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %v: %V", core.Config.SourcePath, err)
	}

	return &Jobs{
		Pool:        make(chan *Job),
		TargetFiles: target,
		SourceFiles: source,
	}, nil
}

func (j *Jobs) GenerateJobs() {
	defer close(j.Pool)
	for _, targetFile := range j.TargetFiles {
		for _, sourceFile := range j.SourceFiles {
			targetArr := strings.Split(targetFile.Name(), ".")
			sourceArr := strings.Split(sourceFile.Name(), ".")
			if targetArr[1] == sourceArr[1] {
				fullPathTarget := core.Config.TargetPath + "/" + targetFile.Name()
				fullPathSource := core.Config.SourcePath + "/" + sourceFile.Name()
				j.Pool <- &Job{Target: fullPathTarget, Source: fullPathSource, TargetName: targetFile.Name(), SourceName: sourceFile.Name()}
			}
		}
	}
}
