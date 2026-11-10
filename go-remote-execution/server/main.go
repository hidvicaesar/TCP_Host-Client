package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os/exec"
	"runtime"
	"sync/atomic"
	"time"
)

// Counter untuk memberikan ID unik setiap client
var clientCounter uint64

// Struktur permintaan dari client
type Request struct {
	Command string `json:"command"`
}

// Struktur jawaban dari server
type Response struct {
	Command string `json:"command"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// Menjalankan perintah yang diizinkan
func executeCommand(command string) Response {
	response := Response{
		Command: command,
	}

	var program string

	switch command {
	case "hostname":
		program = "hostname"

	case "whoami":
		program = "whoami"

	case "ipconfig":
		if runtime.GOOS != "windows" {
			response.Error = "Perintah ini khusus Windows"
			return response
		}
		program = "ipconfig"

	case "systeminfo":
		if runtime.GOOS != "windows" {
			response.Error = "Perintah ini khusus Windows"
			return response
		}
		program = "systeminfo"

	default:
		response.Error = "Perintah tidak diizinkan"
		return response
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	cmd := exec.CommandContext(ctx, program)
	output, err := cmd.CombinedOutput()

	response.Output = string(output)

	if err != nil {
		response.Error = err.Error()
	}

	return response
}

// Menangani koneksi setiap client
func handleConnection(conn net.Conn, clientID uint64) {
	defer conn.Close()

	clientName := fmt.Sprintf("CLIENT %d", clientID)

	log.Printf("[%s] Terhubung | IP: %s",
		clientName, conn.RemoteAddr())

	defer log.Printf("[%s] Terputus", clientName)

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var request Request

		err := decoder.Decode(&request)
		if err != nil {
			return
		}

		log.Printf("[%s] Perintah diterima: %s",
			clientName, request.Command)

		response := executeCommand(request.Command)

		if err := encoder.Encode(response); err != nil {
			log.Printf("[%s] Gagal mengirim hasil: %v",
				clientName, err)
			return
		}

		log.Printf("[%s] Hasil berhasil dikirim",
			clientName)
	}
}

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	fmt.Println("========================================")
	fmt.Println("       GO REMOTE EXECUTION SERVER")
	fmt.Println("========================================")
	fmt.Println("Server berjalan di 127.0.0.1:8080")
	fmt.Println("Menunggu koneksi client...")
	fmt.Println()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Gagal menerima koneksi: %v", err)
			continue
		}

		// Membuat ID unik untuk setiap koneksi
		clientID := atomic.AddUint64(&clientCounter, 1)

		// Menangani client menggunakan goroutine
		go handleConnection(conn, clientID)
	}
}
