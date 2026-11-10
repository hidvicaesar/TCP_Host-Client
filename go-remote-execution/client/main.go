package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
)

// Struktur permintaan
type Request struct {
	Command string `json:"command"`
}

// Struktur jawaban
type Response struct {
	Command string `json:"command"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

func main() {
	fmt.Println("================================")
	fmt.Println(" GO REMOTE EXECUTION CLIENT")
	fmt.Println("================================")

	// Menghubungkan client ke server
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println("Gagal terhubung ke server:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Berhasil terhubung ke server!")
	fmt.Println()

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Perintah tersedia:")
	fmt.Println("- hostname")
	fmt.Println("- whoami")
	fmt.Println("- ipconfig")
	fmt.Println("- systeminfo")
	fmt.Println("- exit")
	fmt.Println()

	for {
		fmt.Print("Masukkan perintah: ")

		if !scanner.Scan() {
			break
		}

		command := scanner.Text()

		if command == "exit" {
			fmt.Println("Client ditutup.")
			break
		}

		// Mengirim permintaan ke server
		request := Request{
			Command: command,
		}

		if err := encoder.Encode(request); err != nil {
			fmt.Println("Gagal mengirim perintah:", err)
			break
		}

		// Menerima hasil dari server
		var response Response

		if err := decoder.Decode(&response); err != nil {
			fmt.Println("Gagal menerima hasil:", err)
			break
		}

		fmt.Println()
		fmt.Println("========== HASIL SERVER ==========")
		fmt.Println("Perintah:", response.Command)

		if response.Error != "" {
			fmt.Println("Error:", response.Error)
		}

		fmt.Println("Output:")
		fmt.Println(response.Output)
		fmt.Println("==================================")
		fmt.Println()
	}
}
