package httptrace

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

var jsonFormatter = log.JSONFormatter{
	TimestampFormat: time.RFC3339,
	FieldMap: log.FieldMap{
		log.FieldKeyTime: "timestamp",
	},
}

type wrappFormatter struct {
	service string
}

var formatter = &wrappFormatter{}

// Format formats the log entry in JSON. It also adds `service` key which contains the
// name of the service. This is useful to distinguish logs per service when you have many
// different services.
// The `timestamp` contains the UTC time in `time.RFC3339` format. Message of the log is
// contained in `msg` key.
func (f wrappFormatter) Format(entry *log.Entry) ([]byte, error) {
	e := entry.WithFields(log.Fields{
		"service": f.service,
	})

	e.Time = time.Now().UTC()
	e.Level = entry.Level
	e.Message = entry.Message
	return (&jsonFormatter).Format(e)
}

func init() {
	formatter.service = os.Getenv("SERVICE_NAME")
	log.SetFormatter(formatter)
	log.SetOutput(os.Stdout)
}
