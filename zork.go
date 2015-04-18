package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Zork struct {
	Cmd    *exec.Cmd
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
}

func NewZork(dfrotz, zork1Dat string) (z *Zork, err error) {
	z = new(Zork)
	z.Cmd = exec.Command(dfrotz, zork1Dat)
	z.Stdin, err = z.Cmd.StdinPipe()
	if err != nil {
		return
	}
	z.Stdout, err = z.Cmd.StdoutPipe()
	if err != nil {
		return
	}
	return
}

func (z *Zork) Run() error {
	if err := z.Cmd.Start(); err != nil {
		log.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- z.Cmd.Wait()
	}()

	input := readFrom(os.Stdin, '\n')
	output := readFrom(z.Stdout, '>')
	for {
		select {
		case s := <-input:
			z.HandleInput(s)
		case s := <-output:
			z.HandleOutput(s)
		case err := <-done:
			return err
		}
	}
}

func (z *Zork) Input(s string) {
	z.Stdin.Write([]byte(s + "\n"))
}

func (z *Zork) HandleInput(s string) {
	s = strings.TrimSpace(s)
	z.Input(s)
}

func (z *Zork) HandleOutput(s string) {
	lines := strings.Split(s, "\n")
	// Determine if there is a header or not If there is not a header,
	// than a move was not completed.
	headerFields := strings.Fields(lines[0])
	n := len(headerFields)
	if n < 5 || headerFields[n-4] != "Score:" || headerFields[n-2] != "Moves:" {
		return
	}
	locationName := strings.Join(headerFields[:n-4], " ")
	// Strip the header, startup info, prompt.
	lines = lines[2:]
	if lines[0] == "ZORK I: The Great Underground Empire" {
		lines = lines[5:]
	}
	lines = lines[:len(lines)-2]
	if lines[0] == locationName {
		// lines now contains the description
	}
	fmt.Print(s)
}

func readFrom(r io.ReadCloser, sentinal byte) chan string {
	ch := make(chan string)
	go func() {
		reader := bufio.NewReader(r)
		for {
			chunk, err := reader.ReadString(sentinal)
			if err == nil {
				ch <- chunk
			}
		}
	}()
	return ch
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <dfrotz> <ZORK1.DAT>\n", os.Args[0])
		os.Exit(2)
	}
	z, err := NewZork(os.Args[1], os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	if err = z.Run(); err != nil {
		log.Fatal(err)
	}
}
