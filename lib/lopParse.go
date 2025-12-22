package lib

import (
	"net"
	"strings"
	"time"
)

// Seviye tanımları (RFC 5424)
const (
	SevEmergency = 0
	SevAlert     = 1
	SevCritical  = 2
	SevError     = 3
	SevWarning   = 4
	SevNotice    = 5
	SevInfo      = 6
	SevDebug     = 7
)

type LogData struct {
	Timestamp  time.Time
	Device     string
	Severity   int8     // Enum8 için 0-7 arası rakam
	Categories []string // Array(String) için slice
	Message    string
}

func ParseLog(messageRaw string, remoteAddr *net.UDPAddr) LogData {
	// Mesajı ilk boşluktan ayır
	parts := strings.SplitN(messageRaw, " ", 2)
	catStr := parts[0]
	message := parts[1]
	if len(parts) < 2 || len(strings.Split(catStr, ",")) < 2 {
		return LogData{
			Device:     remoteAddr.String(),
			Timestamp:  time.Now(),
			Severity:   SevInfo,
			Categories: []string{"unknown"},
			Message:    messageRaw,
		}
	}

	// Severity değeri hariç hepsi categori kapsamındadır.
	topics := strings.Split(catStr, ",")
	categori := make([]string, 0, len(topics)-1)
	categori = append(categori, topics[0])
	categori = append(categori, topics[2:]...)

	// topics[1] esas sev değeridir.
	severity := int8(SevDebug)
	switch topics[1] {
	case "fatal", "emergency":
		severity = SevEmergency
	case "alert":
		severity = SevAlert
	case "critical":
		severity = SevCritical
	case "error":
		severity = SevError
	case "warning":
		severity = SevWarning
	case "notice":
		severity = SevNotice
	case "info":
		severity = SevInfo
	case "debug", "packet", "raw":
		severity = SevDebug
	default:
		severity = SevInfo // Bilinmeyen kategoriler info sayılsın
		categori = append(categori, topics[1])
	}

	return LogData{
		Device:     remoteAddr.String(),
		Timestamp:  time.Now(),
		Severity:   severity,
		Categories: categori,
		Message:    message,
	}
}
