package lib

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func CacheFlush(conn clickhouse.Conn, ch <-chan LogData, CacheTimeout time.Duration, CacheSize int) {
	ticker := time.NewTicker(CacheTimeout)
	var cacheBuffer []LogData
	for {
		select {
		case cacheDatas := <-ch: //Cahnnel toplama
			cacheBuffer = append(cacheBuffer, cacheDatas)
			if len(cacheBuffer) >= CacheSize {
				flushLogs(conn, cacheBuffer)
				cacheBuffer = nil // Buffer'ı sıfırla

			}
		case <-ticker.C:
			// Zaman dolduysa ve içeride veri varsa yaz
			if len(cacheBuffer) > 0 {
				flushLogs(conn, cacheBuffer)
				cacheBuffer = nil
			}
		}
	}
}

// flushLogs: Veriyi fiziksel olarak ClickHouse'a basar
func flushLogs(conn clickhouse.Conn, logs []LogData) {
	// Context timeout ekle ki DB cevap vermezse worker kilitlenmesin
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO mikrotik_logs")
	if err != nil {
		log.Printf("CH Batch Hazırlama Hatası: %v", err)
		return
	}

	for _, l := range logs {
		// lib paketindeki struct yapına göre burayı düzenle
		err := batch.Append(
			l.Timestamp,
			l.Device,
			l.Severity,
			l.Categories,
			l.Message,
		)
		if err != nil {
			log.Printf("Batch Append Hatası: %v", err)
		}
	}

	if err := batch.Send(); err != nil {
		log.Printf("CH Yazma Hatası (Batch Size: %d): %v", len(logs), err)
	} else {
		fmt.Printf(">> %d adet log ClickHouse'a yazıldı.\n", len(logs))
	}
}
