package main

import (
	"fmt"
	"go.bug.st/serial"
)

func main() {
	mode := &serial.Mode{
		BaudRate: 115200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	
	port, err := serial.Open("/dev/ttyUSB0", mode)
	if err != nil {
		fmt.Printf("Failed to open /dev/ttyUSB0: %v\n", err)
		return
	}
	defer port.Close()
	
	fmt.Println("Successfully opened /dev/ttyUSB0")
	
	// Try to read some data
	buf := make([]byte, 16)
	n, err := port.Read(buf)
	if err != nil {
		fmt.Printf("Read error: %v\n", err)
	} else {
		fmt.Printf("Read %d bytes: %x\n", n, buf[:n])
	}
}
