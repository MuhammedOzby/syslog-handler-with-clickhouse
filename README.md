# syslog‑handler‑with‑clickhouse  
> **syslog‑handler‑with‑clickhouse** – UDP üzerinden gelen syslog mesajlarını alır, parçalayıp **ClickHouse** veritabanına toplu olarak yazar.  
> Proje 0‑yazılımı (no‑code) yaklaşımıyla, Go 1.24+ sürümünde derlenebilir.

---

## 📋 Proje Tanıtımı

- **UDP 11514** portu üzerinden Syslog standartı (RFC 5424) ve RFC 3164 ile gelen mesajları dinler.
- Gelen mesajları **LogData** yapısına çevirir (`lib.ParseLog`).
- 1 000 log veya 2 saniyede bir **Cache Flush** tetiklenerek veritabanına toplu yazılır (batch).
- **ClickHouse** bağlantısı için `clickhouse-go/v2` sürücüsü kullanılır.
- Bağlantı havuzu (max 5 bağlantı) ve `godotenv` ile `.env` dosyası okunur.
- Proje **MIT** lisansına sahiptir.

---

## 🚀 Özellikler

| Özellik | Açıklama |
|---------|----------|
| **UDP dinleme** | 11514 portu üzerinden syslog girişi. |
| **Caching** | `CacheSize=1000`, `CacheTimeout=2s` ile toplu yazar. |
| **Veri tabanı** | ClickHouse (SQL engine) – hızlı okuma‑yazma. |
| **Yüksek performans** | Buffer limit (`BufferLimit=10000`) ile burst koruması. |
| **Environment‑based config** | `.env` ile veritabanı, port, IP vb. ayarlanır. |
| **Kolay derleme** | Tek `go build` ile bağımsız binary. |
| **Lisans** | MIT – tam özgürlük. |

---

## 🛠️ Kurulum

> **Önkoşullar**
> - Go 1.24+ (modüler proje, `go.mod` var)
> - ClickHouse sunucusu (örneğin, docker‑de çalışır)

### 1. Depoyu klonlayın

```bash
git clone https://github.com/muhammadsb/syslog-handler-with-clickhouse.git
cd syslog-handler-with-clickhouse
```

### 2. Bağımlılıkları indirin

```bash
go mod download
```

### 3. Çevresel Değişkenleri Ayarlayın

Projenin kök dizininde `.env` dosyası oluşturun:

```env
DB_HOST=clickhouse:9000
DB_NAME=logs
DB_USER=default
DB_PASS=
# Varsayılan: 11514 portu, 0.0.0.0 IP
```

> **Not**  
> `DB_HOST` ClickHouse’ın IP/hostname ve portu (`<ip>:<port>`).  
> ClickHouse’ın `users.xml` dosyasında `logs` veritabanı ve `default` kullanıcı hakları olduğundan emin olun.

### 4. Derleme

```bash
go build -o syslog-collector
```

### 5. Çalıştırma

```bash
./syslog-collector
```

> Çalıştıktan sonra konsolda şu mesajı görürsünüz:  
> `NOC Log Collector dinlemede: 0.0.0.0:11514`

---

## 📦 Örnek Mimari

```
┌───────────────────────┐
│  UDP Dinleyici (11514)│
│  ┌────────────────────┐│
│  │ ParseLog (RFC5424) ││
│  └────────────────────┘│
│  Cache (1k / 2s)      │
│  ┌────────────────────┐│
│  │ CacheFlush         ││
│  └────────────────────┘│
│  ClickHouse Bağlantısı│
└───────────────────────┘
```

> `lib` klasörü içinde:
> - `LogData` yapısı  
> - `ParseLog(msg string) (LogData, error)`  
> - `CacheFlush(ctx context.Context, data []LogData)` (batch insert)

---

## 📦 Kullanım Örneği

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/muhammadsb/syslog-handler-with-clickhouse/lib"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/joho/godotenv"
)

func main() {
	// .env okunuyor
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	// ClickHouse bağlantısı
	conn, err := clickhouse.Open(
		&clickhouse.Options{
			Addr: []string{os.Getenv("DB_HOST")},
			Auth: clickhouse.Auth{
				Database: os.Getenv("DB_NAME"),
				Username: os.Getenv("DB_USER"),
				Password: os.Getenv("DB_PASS"),
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	// Mesaj dinleme
	// (Port/IP varsayılan, lib.ParseLog ile parse)
	// Cache flush işlemi main.go içinde otomatik
}
```

> Günlük tablo şablonu (`logs` veritabanında):

```sql
CREATE TABLE IF NOT EXISTS logs.events (
    id          UInt64,
    host        String,
    appname     String,
    priority    UInt8,
    message     String,
    ts          DateTime
) ENGINE = MergeTree()
ORDER BY ts;
```

> Her 1 000 log veya 2 saniyede bir `INSERT` ile toplu yazım yapılır, böylece ClickHouse’da “write‑latency” düşük kalır.

---

## 🧪 Test

> Projeye birim‑test dosyası eklenmediği için, entegrasyon testleri ClickHouse’la manuel olarak yapılır.  
> Basit bir test için `syslog‑generator` gibi bir araçla UDP üzerinden log gönderin:

```bash
logger -p local0.info "Hello syslog handler" --rfc3164
```

> Çıktının ClickHouse’ta göründüğünden emin olun:

```sql
SELECT * FROM logs.events ORDER BY ts DESC LIMIT 5;
```

---

## 🏗️ Mimari

```
┌─────────────────────┐
│  main.go             │
│  ├─ UDP Dinleyici    │
│  ├─ Cache Flush      │
│  └─ ClickHouse Writer│
└─────────────────────┘
          │
          ▼
┌─────────────────────┐
│  lib/                │
│  ├─ LogData          │
│  ├─ ParseLog()       │
│  └─ CacheFlush()     │
└─────────────────────┘
```

- **Cache Flush**: `go routine` ile zamanlayıcıdır, `CacheSize` ve `CacheTimeout` e göre tetiklenir.
- **Batch**: `conn.WriteBatch` ile tek seferde 1 000 kayıt yazılır.
- **Buffer**: `BufferLimit` 10 000 log’a kadar gelenleri tutar; bu limit aşılırsa `logger.Fatal` ile hata rapor edilir.

---

## 📄 Örnek Yapılandırma

```bash
# .env
DB_HOST=clickhouse:9000
DB_NAME=logs
DB_USER=default
DB_PASS=
```

```sql
-- ClickHouse: logs veritabanı oluşturma
CREATE DATABASE IF NOT EXISTS logs;
USE logs;

-- Tablo oluşturma
CREATE TABLE IF NOT EXISTS events (
    id          UInt64,
    host        String,
    appname     String,
    priority    UInt8,
    message     String,
    ts          DateTime
) ENGINE = MergeTree()
ORDER BY ts;
```

---

## 🙏 Katkıda Bulunma

1. **Fork** ve **branch** oluşturun: `feature/<özellik>`.
2. Değişiklikleri test edin: `go test ./...` (önce test dosyaları eklenmelidir).
3. Commit mesajlarını **Conventional Commits** kuralları ile yazın:
   ```
   feat: cache timeout iyileştirmesi
   fix: ParseLog hatası düzeltildi
   ```
4. Pull‑request gönderin.

---

## 📄 Lisans

MIT. Detaylı bilgi için `LICENSE` dosyasını inceleyin.  
Proje tamamen açık kaynak olup, ticari ve kişisel kullanımda sınırlama yoktur.

--- 

## 📞 Destek

- **Issue**: Herhangi bir hata, öneri ya da sorular için Issues bölümü kullanılabilir.
- **Mail**: muhammadsb@example.com (Opsiyonel)

---

## 📌 Sık Sorulan Sorular

| Soru | Cevap |
|------|-------|
| **Neden ClickHouse?** | Yüksek yazma hızı, kolon‑tabanlı saklama ve sorgu performansı. |
| **UDP 11514 portu** | RFC 5424 (IPv4/IPv6) syslog için yaygın port. |
| **Cache Flush nedir?** | Belirlenen log sayısına/veya süreye ulaşıldığında, cache içindeki verilerin toplu olarak veritabanına yazılması. |
| **Çok sayıda mesaj geldiğinde ne olur?** | `BufferLimit` 10 000, bu değeri aşan mesajlar loglanır ve program sonlandırılır. Bu değer ihtiyaca göre değiştirilebilir. |

---

## 🎉 Katkı Sağlayacaklar

- **Yeni syslog formatları** (`RFC 5424`, `RFC 3164`) için `ParseLog` destek ekleme
- **Dockerfile** ile otomatik container oluşturma
- **Grafana/Prometheus** ile monitoring entegrasyonu
- **CI / CD** pipeline (GitHub Actions, GitLab CI, etc.)

--- 

> Projeyi derledikten sonra `./syslog-collector` çalıştırın ve ClickHouse veritabanınızı izleyin!  
> Teşekkürler!

