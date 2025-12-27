package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"syslog-handler-with-clickhouse/lib"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/joho/godotenv"
)

// Ayarlar
const (
	CacheSize    = 1000            // 1000 log birikince yaz
	CacheTimeout = 2 * time.Second // Veya 2 saniye geçince yaz
	BufferLimit  = 10000           // Kanal kapasitesi (Burst koruması)
)

/*
Ana döngü alanı mesaj geldikçe burada işlemler sağlanacak.
*/
func mainLoop() int { // Logları taşıyacak tamponlu kanal (Buffered Channel)
	var programState int = 0
	portFlag := flag.Int("port", 514, "Deinlenecek UDP port kapı numarası") // go run main.go --port 11514
	flag.Parse()
	// Listener nesnesi
	var ListenerInfo net.UDPAddr = net.UDPAddr{
		Port: *portFlag,
		IP:   net.ParseIP("0.0.0.0"),
	}

	// Gelen mesajlar için buffer
	var messageBuffer = make([]byte, 40960)
	/*-------------------------------------- Clickhouse Conn Başla --------------------------------------*/
	// Bağlantı ayarları
	connClickhouse, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{os.Getenv("DB_HOST")},
		Auth: clickhouse.Auth{
			Database: os.Getenv("DB_NAME"),
			Username: os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASS"),
		},
		// Bağlantı havuzu ayarları (Önemli!)
		MaxOpenConns: 5,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatal("CH Bağlantı Hatası:", err)
	}
	defer connClickhouse.Close()

	// Bağlantıyı test et
	if err := connClickhouse.Ping(context.Background()); err != nil {
		log.Fatal("CH Ping Hatası:", err)
	}
	/*-------------------------------------- Clickhouse Conn Bitir --------------------------------------*/

	// Datayı direk yazarak yormak yerine araya bir cache
	messageCache := make(chan lib.LogData, BufferLimit)

	// INFO: Dinlemeye başla
	Listener, err := net.ListenUDP("udp", &ListenerInfo)
	if err != nil {
		log.Fatalf("Port dinlenemedi: %v", err)
	}
	defer Listener.Close()

	fmt.Printf("NOC Log Collector dinlemede: %s\n", Listener.LocalAddr().String())

	// Cache kanalını sürekli kontrol eden ve boşaltan kısım
	go lib.CacheFlush(connClickhouse, messageCache, CacheTimeout, CacheSize)

	for {
		buffLen, recivedAddr, err := Listener.ReadFromUDP(messageBuffer)
		if err != nil {
			log.Printf("UDP Okuma Hatası: %v", err)
			programState = 1
			break
		}

		messageRaw := string(messageBuffer[0:buffLen])
		messageCache <- lib.ParseLog(messageRaw, recivedAddr)
	}
	return programState
}

/*
Program başlangıç
*/
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(".env dosyası yüklenirken hata oluştu")
	}
	os.Exit(mainLoop())
}
