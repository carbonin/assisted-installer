package ops

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"

	"github.com/eranco74/assisted-installer/generated/bm-inventory/models"
	"github.com/eranco74/assisted-installer/src/inventory_client"
	"github.com/sirupsen/logrus"
)

const MinProgressDelta = 5

type CoreosInstallerLogWriter struct {
	log              *logrus.Logger
	lastLogLine      []byte
	progressReporter inventory_client.InventoryClient
	progressRegex    *regexp.Regexp
	hostID           string
	lastProgress     int
}

func NewCoreosInstallerLogWriter(logger *logrus.Logger, progressReporter inventory_client.InventoryClient, hostID string) *CoreosInstallerLogWriter {
	return &CoreosInstallerLogWriter{log: logger,
		lastLogLine:      []byte{},
		progressReporter: progressReporter,
		progressRegex:    regexp.MustCompile(`^>(.*?)\((.*?)\)\s*\r`),
		hostID:           hostID,
		lastProgress:     0,
	}
}

func (l *CoreosInstallerLogWriter) Write(p []byte) (n int, err error) {
	if bytes.Contains(p, []byte{'\n'}) {
		// If log has a new line - log it
		l.log.Info(string(p))
	} else {
		// Append bytes to last log line slice
		l.lastLogLine = append(l.lastLogLine, p...)
		if bytes.Contains(l.lastLogLine, []byte{'\r'}) {
			// If log contains carriage return - log it and set to empty slice
			l.log.Info(string(l.lastLogLine))
			l.reportProgress()
			l.lastLogLine = []byte{}

		}
	}
	return len(p), nil
}

func (l *CoreosInstallerLogWriter) reportProgress() {
	match := l.progressRegex.FindStringSubmatch(string(l.lastLogLine))
	if len(match) < 3 {
		return
	}
	currentPercent, err := strconv.Atoi(strings.TrimRight(match[2], "%"))
	// in case we fail to parse the log line we do nothing
	if err != nil {
		return
	}
	if currentPercent >= l.lastProgress+MinProgressDelta {
		// If the progress is more than 5% report it
		if err := l.progressReporter.UpdateHostInstallProgress(l.hostID, models.HostStageWritingImageToDisk, match[2]); err == nil {
			l.lastProgress = currentPercent
		}
	}
}
