package log

type PrefixLogger struct {
	prefix string
}

func ProducePrefixLogger(prefix string) *PrefixLogger {
	return &PrefixLogger{prefix: prefix}
}

func (p *PrefixLogger) V1(format string, v ...interface{}) {
	V1(p.prefix+format, v...)
}

func (p *PrefixLogger) V2(format string, v ...interface{}) {
	V2(p.prefix+format, v...)
}

func (p *PrefixLogger) V5(format string, v ...interface{}) {
	V5(p.prefix+format, v...)
}

func (p *PrefixLogger) Info(format string, v ...interface{}) {
	Info(p.prefix+format, v...)
}
