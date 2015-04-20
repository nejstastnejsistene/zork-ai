package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// How long of a timeout should indicate the end of input.
const timeout = 10 * time.Millisecond

type Zork struct {
	Cmd        *exec.Cmd   // The zork process.
	Error      chan error  // Channel for errors or exit status.
	ZorkInput  io.Writer   // Zork's stdin.
	ZorkOutput chan string // Stream of output from
	Stdin      chan string // Stream of input from stdin.
	Mutex      sync.Mutex  // Mutex to protect
}

func RunZork(dfrotz, zork1Dat string) (err error) {
	z := new(Zork)
	z.Error = make(chan error, 2)
	z.Cmd = exec.Command(dfrotz, zork1Dat)
	// Pipes to process.
	z.ZorkInput, err = z.Cmd.StdinPipe()
	if err != nil {
		return
	}
	var stdout io.ReadCloser
	stdout, err = z.Cmd.StdoutPipe()
	if err != nil {
		return
	}
	// Start process.
	if err := z.Cmd.Start(); err != nil {
		log.Fatal(err)
	}
	// Wait for process to exit.
	go func() {
		z.Error <- z.Cmd.Wait()
	}()
	// Kill the process when finished if it isn't already dead
	defer func() {
		// Signal 0 checks if the process is alive.
		if z.Cmd.Process.Signal(syscall.Signal(0)) == nil {
			z.Cmd.Process.Kill()
		}
	}()
	// Channels to read data from zork's stdout and our stdin.
	z.ZorkOutput = sepByTimeout(stdout, timeout)
	z.Stdin = sepByTimeout(os.Stdin, timeout)

	// Handle initial output.
	z.HandleAsync("", <-z.ZorkOutput)
	// Main loop.
	for {
		// Read input from stdin (excepting errors).
		select {
		case input := <-z.Stdin:
			// Pipe the input to zork and read the output.
			input = strings.TrimSpace(input)
			output, err := z.EvaluateCommand(input)
			if err != nil {
				return err
			}
			// Process the input/output pair.
			go z.Handle(input, output)
		case err := <-z.Error:
			return err
		}
	}
}

// Returns a channel that yields all data received by a reader only after
// a timeout is reached. Needed for zork because there is never an EOF, and you
// ordinarly only know the end of input by visually seeing that no more input
// is coming.
func sepByTimeout(r io.ReadCloser, timeout time.Duration) chan string {
	// Continuosly read chunks.
	chunks := make(chan []byte)
	go func() {
		chunk := make([]byte, 4096)
		for {
			n, err := r.Read(chunk)
			// End input on error or if nothing is read.
			if n == 0 || err != nil {
				r.Close()
				close(chunks)
				return
			}
			chunks <- chunk[:n]
		}
	}()
	// Accumulate chunks until the timeout is reached.
	result := make(chan string)
	var buf []byte
	go func() {
		for {
			select {
			// Another chunk, timeout not reached yet.
			case chunk, ok := <-chunks:
				// End of input.
				if !ok {
					close(result)
					return
				}
				buf = append(buf, chunk...)
			// Timeout occurs before next data, yield accumulated buffer.
			case <-time.After(timeout):
				if len(buf) > 0 {
					result <- string(buf)
					buf = nil
				}
			}
		}
	}()
	return result
}

// Input a command (excluding newline) into zork and return its output.
func (z *Zork) EvaluateCommand(command string) (output string, err error) {
	// Write the input (plus a newline) to zork.
	if _, err = z.ZorkInput.Write([]byte(command + "\n")); err != nil {
		return
	}
	// Read the output from zork, excepting the process dying.
	select {
	case output = <-z.ZorkOutput:
		return
	case err = <-z.Error:
		return
	}
}

func (z *Zork) HandleAsync(input, output string) {
	if err := z.Handle(input, output); err != nil {
		z.Error <- err
	}
}

func (z *Zork) Handle(input, output string) (err error) {
	// Only let this run once at a time.
	z.Mutex.Lock()
	defer z.Mutex.Unlock()
	// Print the output like it usually would.
	fmt.Print(output)
	os.Stdout.Sync()

	// Save to a random file, as demo of what can be done!
	//b := make([]byte, 8)
	//rand.Read(b)
	//name := hex.EncodeToString(b)
	//if err = z.Save(name+".sav", false); err != nil {
	//	return
	//}

	/*lines := strings.Split(output, "\n")
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
	}*/
	return
}

func (z *Zork) Save(path string, overwrite bool) (err error) {
	var output string
	_, err = z.EvaluateCommand("save")
	if err != nil {
		return nil
	}
	output, err = z.EvaluateCommand(path)
	if err != nil {
		return
	}
	if output == "Overwrite existing file? " {
		if !overwrite {
			return errors.New("save file already exists")
		}
		output, _ = z.EvaluateCommand("y")
		if err != nil {
			return
		}
	}
	if !strings.HasPrefix(output, "Ok") {
		err = errors.New(output)
	}
	return
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: %s <dfrotz> <ZORK1.DAT>\n", os.Args[0])
	}
	if err := RunZork(os.Args[1], os.Args[2]); err != nil {
		log.Fatal(err)
	}
}
