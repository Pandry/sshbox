package main

import (
	"fmt"
	"log"

	"github.com/gliderlabs/ssh"
	// github.com/spf13/viper
)

func main() {
	ssh.Handle(func(s ssh.Session) {
		//io.WriteString(s, "Hello world\n")
		defer s.Close()
		localText := ""
		for { // While(true)
			buffer := make([]byte, 1)
			n, err := s.Read(buffer)
			if err != nil {
				s.Write([]byte(err.Error()))
				continue
			}
			if n == 0 {
				continue
			}
			fmt.Println("Received: ", buffer[0])
			s.Write(buffer)
			if buffer[0] == '\r' {
				s.Write([]byte("\n"))
				s.Write([]byte(localText + "\r\n"))
				localText = ""
			} else {
				localText += string(buffer)
			}
		}
	})

	log.Fatal(ssh.ListenAndServe(":2222", nil))

}
