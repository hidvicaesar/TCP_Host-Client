# Go Concurrent TCP Remote Execution Engine

![Go Version](https://img.shields.io/badge/Go-1.27.3-00ADD8?style=flat&logo=go)
![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?style=flat&logo=windows)
![Protocol](https://img.shields.io/badge/Protocol-TCP%20%2F%20JSON-orange)
![Concurrency](https://img.shields.io/badge/Concurrency-Goroutines%20%2B%20Atomic-brightgreen)

Implementasi sistem _client-server_ berbasis _Transmission Control Protocol_ (TCP) menggunakan bahasa pemrograman Go dengan arsitektur multi-koneksi konkuren (_goroutine-driven_) untuk mengeksekusi perintah diagnostik sistem operasi secara jarak jauh (_remote execution_). Sistem dirancang khusus untuk eksekusi _native_ pada lingkungan Windows dengan proteksi _command whitelisting_, pembatasan batas waktu eksekusi (_timeout_), serta pertukaran data terstruktur berformat JSON.

---

## 📐 Arsitektur Sistem

Sistem menggunakan model konkurensi M:N berbasis _Goroutine_. _Main thread server_ bertindak sebagai _listener_ pasif yang mendelegasikan setiap sesi koneksi masuk ke _worker goroutine_ terpisah, sehingga mampu melayani banyak _client_ secara simultan tanpa _blocking_.

```mermaid
graph TD
    subgraph Clients ["Client Instances (Localhost)"]
        C1["Client 1\n(CLI Terminal)"]
        C2["Client 2\n(CLI Terminal)"]
        C3["Client 3\n(CLI Terminal)"]
    end

    subgraph Network ["TCP Layer"]
        TCP["TCP Socket Stream\n127.0.0.1:8080\nPayload: JSON"]
    end

    subgraph ServerEngine ["Go Server Engine"]
        LST["TCP Listener\n(net.Listen)"]
        ATM["Atomic Counter\n(sync/atomic.AddUint64)"]

        subgraph Workers ["Concurrent Goroutines"]
            G1["Goroutine Worker 1\n[CLIENT 1]"]
            G2["Goroutine Worker 2\n[CLIENT 2]"]
            G3["Goroutine Worker 3\n[CLIENT 3]"]
        end

        subgraph Security ["Security & Execution Pipeline"]
            VAL{"Command Whitelist\n& OS Validation"}
            CTX["context.WithTimeout\n(10s Limit)"]
            EXEC["os/exec.CommandContext\n(CombinedOutput)"]
        end
    end

    subgraph OS ["Host Operating System"]
        WIN["Windows Subsystems / Binaries\n(hostname / whoami / ipconfig / systeminfo)"]
    end

    C1 ==>|net.Dial| TCP
    C2 ==>|net.Dial| TCP
    C3 ==>|net.Dial| TCP
    TCP ==> LST
    LST --> ATM --> Workers
    G1 -.-> VAL
    G2 -.-> VAL
    G3 -.-> VAL
    VAL -->|Valid & Allowed| CTX --> EXEC --> WIN
    VAL -->|Unauthorized| ERR["Error: Perintah tidak diizinkan"]
    WIN -->|stdout / stderr| EXEC
    EXEC --> Workers
    ERR --> Workers
    Workers ==>|JSON Response| TCP ==> Clients
```

---

## 🔄 Alur Komunikasi & Protokol (Sequence Diagram)

Pertukaran data memanfaatkan abstraksi _streaming_ `json.Encoder` dan `json.Decoder` langsung di atas aliran _socket_ TCP (`net.Conn`):

```mermaid
sequenceDiagram
    autonumber
    participant C as Client Instance
    participant S as Server (Listener 8080)
    participant G as Client Goroutine Worker
    participant E as os/exec Subprocess

    Note over C,S: Inisiasi Sesi Jaringan
    C->>S: TCP Three-Way Handshake (SYN, SYN-ACK, ACK)
    S->>G: Spawn Goroutine via 'go handleConnection(conn, id)'
    Note right of S: ID unik via atomic.AddUint64(&clientCounter, 1)

    Note over C,G: Siklus Request-Response
    loop Interactive CLI Loop
        C->>G: JSON Request: {"command": "hostname"}
        Note over G: Log: [CLIENT X] Perintah diterima: hostname
        G->>G: Validasi Whitelist & runtime.GOOS
        alt Perintah Valid
            G->>E: exec.CommandContext(ctx, "hostname")
            E-->>G: CombinedOutput (stdout / stderr)
            G->>C: JSON Response: {"command":"hostname","output":"DESKTOP-ABC\n"}
        else Perintah Ilegal
            G->>C: JSON Response: {"command":"dir","output":"","error":"Perintah tidak diizinkan"}
        end
    end

    Note over C,G: Terminasi Sesi
    C->>G: Input "exit" / Close TCP FIN
    G->>G: defer conn.Close()
    Note over G: Log: [CLIENT X] Terputus
```

---

## ✨ Fitur Utama

- **Concurrent Multi-Client Handling:** Menggunakan _goroutine_ ringan untuk melayani koneksi banyak _client_ secara paralel tanpa membebani sumber daya sistem operasi.
- **Thread-Safe Client Identification:** Menggunakan `sync/atomic.AddUint64` untuk menjamin keunikan penomoran ID _client_ (`CLIENT 1`, `CLIENT 2`, dst.) bebas dari kondisi _race condition_.
- **Command Whitelisting:** Perlindungan terhadap ancaman _remote arbitrary command execution_ dengan membatasi eksekusi hanya pada perintah yang terdaftar.
- **Execution Timeout Safeguard:** Pembatasan waktu proses sub-sistem selama maksimal 10 detik via `context.WithTimeout` guna mencegah proses macet (_hang_).
- **OS-Specific Guardrail:** Pengecekan otomatis variabel `runtime.GOOS` untuk memastikan utilitas diagnostik Windows (`ipconfig` dan `systeminfo`) hanya dieksekusi pada lingkungan yang sesuai.
- **Standardized JSON Framing:** Menggunakan objek `Request` dan `Response` terstruktur dengan tag `omitempty` untuk meminimalkan ukuran transmisi data.

---

## 📂 Struktur Direktori Proyek

```text
go-remote-execution/
├── client/
│   └── main.go       # Program client CLI, handler socket, dan parser JSON
├── server/
│   └── main.go       # Listener TCP, manajemen goroutine, dan executor perintah
└── go.mod            # File konfigurasi modul Go (module remote-execution)
```

---

## 📋 Daftar Perintah yang Diizinkan

| Perintah     | Target Eksekusi  | Batasan Platform | Deskripsi Fungsional                                                          |
| :----------- | :--------------- | :--------------- | :---------------------------------------------------------------------------- |
| `hostname`   | `hostname.exe`   | Lintas Platform  | Mengambil dan menampilkan nama _host_ komputer _server_.                      |
| `whoami`     | `whoami.exe`     | Lintas Platform  | Menampilkan identitas akun pengguna (_username_) Windows aktif pada _server_. |
| `ipconfig`   | `ipconfig.exe`   | `windows` only   | Menampilkan konfigurasi antarmuka jaringan dan alamat IPv4/IPv6 _server_.     |
| `systeminfo` | `systeminfo.exe` | `windows` only   | Menampilkan rincian konfigurasi sistem operasi dan spesifikasi perangkat.     |
| `exit`       | _Internal CLI_   | Lintas Platform  | Memutuskan sesi koneksi TCP dan menutup program _client_.                     |

> **Catatan Keamanan:** Perintah di luar daftar di atas (misalnya `dir`, `rmdir`, `powershell`, atau upaya penyisipan operator chaining `&&`, `|`) akan otomatis ditolak dengan pesan galat `"Perintah tidak diizinkan"`.

---

## 🚀 Panduan Instalasi & Penggunaan

### Prasyarat Lingkungan

- Sistem Operasi: Windows 10 / 11 (64-bit)
- [Go Compiler](https://go.dev/dl/) versi `1.27.3` atau lebih baru
- Terminal Windows PowerShell atau Command Prompt

### 1. Kloning Repositori

```powershell
git clone https://github.com/hidvicaesar/TCP_Host-Client.git
cd TCP_Host-Client/go-remote-execution
```

### 2. Menjalankan Server

Buka terminal PowerShell pertama dan jalankan perintah:

```powershell
go run ./server/main.go
```

_Output Terminal Server:_

```text
========================================
       GO REMOTE EXECUTION SERVER
========================================
Server berjalan di 127.0.0.1:8080
Menunggu koneksi client...
```

### 3. Menjalankan Client

Buka jendela terminal PowerShell kedua dan jalankan perintah:

```powershell
go run ./client/main.go
```

_Output Terminal Client:_

```text
================================
 GO REMOTE EXECUTION CLIENT
================================
Berhasil terhubung ke server!

Perintah tersedia:
- hostname
- whoami
- ipconfig
- systeminfo
- exit

Masukkan perintah: hostname

========== HASIL SERVER ==========
Perintah: hostname
Output:
DESKTOP-TELECOM

==================================
```

---

## 🧪 Pengujian Konkurensi (3 Client dalam 1 Host)

Pengujian dilakukan untuk membuktikan ketahanan _server_ saat menerima koneksi simultan dari tiga _instance client_ yang berjalan bersamaan di alamat _loopback_ `127.0.0.1:8080`:

```text
┌─────────────────────────────────┐   ┌─────────────────────────────────┐
│       TERMINAL: CLIENT 1        │   │       TERMINAL: CLIENT 2        │
│ Masukkan perintah: whoami       │   │ Masukkan perintah: hostname     │
│ [Output: DESKTOP-PC\Engineer]   │   │ [Output: DESKTOP-PC]            │
└─────────────────────────────────┘   └─────────────────────────────────┘
                 ▲                                     ▲
                 │                                     │
                 └──────────────────┬──────────────────┘
                                    │
                                    ▼
┌───────────────────────────────────────────────────────────────────────┐
│                           TERMINAL: SERVER                            │
│ 2026/11/10 10:00:01 [CLIENT 1] Terhubung | IP: 127.0.0.1:52140        │
│ 2026/11/10 10:00:03 [CLIENT 2] Terhubung | IP: 127.0.0.1:52145        │
│ 2026/11/10 10:00:05 [CLIENT 3] Terhubung | IP: 127.0.0.1:52151        │
│ 2026/11/10 10:00:10 [CLIENT 1] Perintah diterima: whoami              │
│ 2026/11/10 10:00:10 [CLIENT 1] Hasil berhasil dikirim                 │
│ 2026/11/10 10:00:12 [CLIENT 2] Perintah diterima: hostname            │
│ 2026/11/10 10:00:12 [CLIENT 2] Hasil berhasil dikirim                 │
└───────────────────────────────────────────────────────────────────────┘
                                    ▲
                                    │
┌───────────────────────────────────┴───────────────────────────────────┐
│                          TERMINAL: CLIENT 3                           │
│ Masukkan perintah: ipconfig                                           │
│ [Output: Windows IP Configuration - Ethernet adapter & IPv4 details]  │
└───────────────────────────────────────────────────────────────────────┘
```

### Hasil Analisis Konkurensi

1. **Pemisahan Port Sumber:** Setiap _client_ dialokasikan nomor _ephemeral port_ independen oleh sistem operasi Windows (misal: `52140`, `52145`, `52151`), mencegah tubrukan paket data.
2. **Isolasi Eksekusi:** Perintah dengan waktu proses tinggi (`systeminfo` atau `ipconfig`) pada satu _client_ tidak memicu kelambatan (_head-of-line blocking_) bagi _client_ lain yang menjalankan perintah cepat (`hostname`).
3. **Atomic Counting:** Identifikasi terminal terurut secara deterministik (`CLIENT 1`, `CLIENT 2`, `CLIENT 3`) tanpa bentrok nilai variabel bersama.

---

## 📦 Spesifikasi Skema Data (JSON Payload)

### Request (Client -> Server)

```json
{
  "command": "whoami"
}
```

### Response (Server -> Client - Berhasil)

```json
{
  "command": "whoami",
  "output": "desktop-telecom\\student\r\n"
}
```

### Response (Server -> Client - Ditolak)

```json
{
  "command": "dir",
  "output": "",
  "error": "Perintah tidak diizinkan"
}
```

---

## 👤 Penulis

- **Nama:** Hidvicaesar Ayangfaana Hafidz
- **NIM:** 4.31.23.1.10
- **Kelas / No. Absen:** TE-4B / 09
- **Program Studi:** Sarjana Terapan Teknik Telekomunikasi, Politeknik Negeri Semarang
