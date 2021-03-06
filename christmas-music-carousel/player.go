package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

func playforever(midiport string, files []string, wg *sync.WaitGroup, quit <-chan struct{}) <-chan error {
	err := make(chan error)

	wg.Add(1)
	go func() {
		defer Debug.Println("Player watcher stopped")
		defer wg.Done()
		defer close(err)

		// play indefinitly the list of songs
		for {
			var lasterror error
			readOneMusic := false
			for _, f := range files {
				start := time.Now()
				lasterror = aplaymidi(midiport, f, quit)
				end := time.Now()

				// check for quitting request
				select {
				case <-quit:
					Debug.Println("Quit player watcher as requested")
					return
				default:
				}

				if end.Sub(start) > time.Duration(time.Second) {
					readOneMusic = true
				}
			}

			// exit loop if we couldn't play any music
			if !readOneMusic {
				if lasterror != nil {
					err <- lasterror
					return
				}
				err <- errors.New("aplaymidi fails playing any files")
				return
			}
		}
	}()

	return err
}

// run aplaymidi on listed file path.
func aplaymidi(midiport string, filename string, quit <-chan struct{}) error {
	Debug.Printf("Playing %s", filename)
	cmd := exec.Command("aplaymidi", "-p", midiport, filename)
	var errbuf bytes.Buffer
	cmd.Stderr = &errbuf
	// prevent Ctrl + C and other signals to get sent
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := cmd.Start()
	if err != nil {
		return err
	}

	// killer goroutine
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-quit:
			Debug.Println("Forcing aplaymidi to stop")
			cmd.Process.Kill()
		case <-done:
		}
	}()

	e := cmd.Wait()
	if e != nil {
		e = fmt.Errorf("%s: %v", errbuf.String(), e)
	}
	return e
}
