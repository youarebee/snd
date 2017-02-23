package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"os/exec"

	"log"

	"github.com/davecheney/gpio"
)

var buttons = []int{gpio.GPIO23, gpio.GPIO24, gpio.GPIO25}

type Event struct {
	Pin  int
	High bool
}

var players map[int]*exec.Cmd

func main() {
	players = make(map[int]*exec.Cmd)
	done := make(chan struct{}, 1)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			fmt.Println("Closing pin and terminating program.")
			done <- struct{}{}
		}
	}()

	eventChan := make(chan Event, 100)
	defer close(eventChan)

	go processEvent(eventChan)

	for i, b := range buttons {
		pin, err := gpio.OpenPin(b, gpio.ModeInput)
		if err != nil {
			fmt.Printf("Error opening pin! %s\n", err)
			return
		}
		defer pin.Close()

		index := i
		err = pin.BeginWatch(gpio.EdgeBoth, func() {
			eventChan <- Event{Pin: index, High: pin.Get()}
		})
		if err != nil {
			fmt.Printf("Unable to watch rising pin: %s\n", err.Error())
			return
		}

		defer pin.EndWatch()

	}

	<-done
}

var (
	state [3]bool
)

/*
0 זה עמוד אחד.
1 זה עמוד 2
2 זה שני העמודים ביחד אבל בלי קצר
3 זה שרשרת בין שני העמודים
*/
func processEvent(eventChan chan Event) {

	for e := range eventChan {
		state[e.Pin] = !e.High

		switch {
		case state[2]:
			stop(0)
			stop(1)
			stop(2)
			play(3)
		case state[1] && state[0]:

			stop(0)
			stop(1)
			play(2)
			stop(3)
		case state[1]:

			stop(0)
			play(1)
			stop(2)
			stop(3)
		case state[0]:

			play(0)
			stop(1)
			stop(2)
			stop(3)
		default:

			stop(0)
			stop(1)
			stop(2)
			stop(3)
		}
	}
}

func play(i int) {
	log.Print("play called ", i)

	if oldcmd, ok := players[i]; ok {
		err := oldcmd.Process.Signal(syscall.Signal(0))
		if err != nil {
			oldcmd.Process.Release()
			delete(players, i)
		} else {
			// already playing, nothing to do
			// probably bug?
			return
		}
	}

	arr := make([]string, 0, 50)
	s := fmt.Sprintf("%d.mp3", i)
	for i := 0; i < cap(arr); i++ {
		arr = append(arr, s)
	}

	cmd := exec.Command("mpg123", arr...)
	cmd.Start()
	players[i] = cmd
}

func stop(i int) {
	log.Print("stop called ", i)
	if oldcmd, ok := players[i]; ok {
		oldcmd.Process.Kill()
		oldcmd.Process.Release()
		delete(players, i)
	}

}
