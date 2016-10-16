// Copyright (c) 2016 Josh Gardiner aka NeuralSpaz on github.com
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// example usage of the basic xethru protocol respiration app
package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/NeuralSpaz/xethru"
	"github.com/gorilla/websocket"
	"github.com/jacobsa/go-serial/serial"
)

var resetflag bool

func main() {
	log.Println("X2M200 Web Demo")

	commPort := flag.String("com", "/dev/ttyACM0", "the comm port you wish to use")
	baudrate := flag.Uint("baud", 115200, "the baud rate for the comm port you wish to use")
	sensitivity := flag.Uint("sensitivity", 7, "the sensitivity")
	start := flag.Float64("start", 0.5, "start of detection zone")
	end := flag.Float64("end", 2.1, "end of detection zone")
	listen := flag.String("listen", "127.0.0.1:23000", "host:port to start webserver")
	reset := flag.Bool("reset", false, "try to reset the sensor")
	// format := flag.String("format", "json", "format for the log files valid choices are csv and json")
	flag.Parse()
	resetflag = *reset

	time.Sleep(time.Second * 1)
	baseband := make(chan xethru.BaseBandAmpPhase)
	resp := make(chan xethru.Respiration)
	sleep := make(chan xethru.Sleep)
	url := "http://" + *listen
	go openXethru(*commPort, *baudrate, *sensitivity, *start, *end, url, baseband, resp, sleep)

	// initize maps of active websocket connections
	baseBandconnections = make(map[*websocket.Conn]bool)
	respirationconnections = make(map[*websocket.Conn]bool)
	sleepconnections = make(map[*websocket.Conn]bool)

	http.HandleFunc("/ws/bb", baseBandwsHandler)
	http.HandleFunc("/ws/resp", respirationwsHandler)
	http.HandleFunc("/ws/sleep", sleepwsHandler)

	// http.Handle("/", http.FileServer(http.Dir("./www")))
	http.HandleFunc("/js/reconnecting-websocket.min.js", websocketReconnectHandler)
	http.HandleFunc("/", indexHandler)

	// start webserver in the background
	go func() {
		err := http.ListenAndServe(*listen, nil)
		if err != nil {
			log.Panic(err)
		}
	}()

	// open default browser

	date := time.Now().Format(time.RFC822)
	respirationfile, err := os.Create("./resp " + date + ".json")
	if err != nil {
		log.Println(err)
	}
	respirationenc := json.NewEncoder(respirationfile)

	sleepfile, err := os.Create("./sleep " + date + ".json")
	if err != nil {
		log.Println(err)
	}
	sleepenc := json.NewEncoder(sleepfile)

	for {
		select {
		case data := <-baseband:
			b, err := json.Marshal(data)
			if err != nil {
				log.Panicln("Error Marshaling: ", err)
			}
			go sendBaseBand(b)
		case data := <-resp:
			b, err := json.Marshal(data)
			if err != nil {
				log.Panicln("Error Marshaling: ", err)
			}
			go sendrespiration(b)
			if err := respirationenc.Encode(&data); err != nil {
				log.Println(err)
			}
		case data := <-sleep:
			b, err := json.Marshal(data)
			if err != nil {
				log.Panicln("Error Marshaling: ", err)
			}
			go sendsleep(b)
			if err := sleepenc.Encode(&data); err != nil {
				log.Println(err)
			}
		}
	}
}

// open default browser with url
func open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func resetSensor(comm string, baudrate uint) error {
	// c := &serial.Config{Name: comm, Baud: int(baudrate)}
	options := serial.OpenOptions{
		PortName:        comm,
		BaudRate:        baudrate,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 4,
	}

	port, err := serial.Open(options)
	if err != nil {
		log.Printf("serial.Open: %v\n", err)
	}
	// port.Flush()

	x2 := xethru.Open("x2m200", port)
	// defer port.Close()
	defer x2.Close()

	reset, err := x2.Reset()
	if err != nil {
		log.Printf("serial.Reset: %v\n", err)
		return err
	}
	if !reset {
		log.Fatal("Could not reset")
	}
	return nil
}

func openXethru(comm string, baudrate uint, sensivity uint, start float64, end float64, url string, baseband chan xethru.BaseBandAmpPhase, resp chan xethru.Respiration, sleep chan xethru.Sleep) {

	if resetflag {
		err := resetSensor(comm, baudrate)
		if err != nil {
			log.Panic(err)
		}
	}

	count := 5
	for {
		select {
		case <-time.After(time.Second):
			count--
			log.Println("Waiting for sensor " + strconv.Itoa(count))
		}
		if count <= 0 {
			break
		}
	}

	options := serial.OpenOptions{
		PortName:        comm,
		BaudRate:        baudrate,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 4,
	}

	port, err := serial.Open(options)

	// c := &serial.Config{Name: comm, Baud: int(baudrate)}
	// port, err := serial.OpenPort(c)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}

	x2 := xethru.Open("x2m200", port)
	defer x2.Close()

	m := xethru.NewModule(x2, "sleep")

	log.Printf("%#+v\n", m)
	err = m.Load()
	if err != nil {
		log.Panicln(err)
	}

	log.Println("Setting LED MODE")
	m.LEDMode = xethru.LEDInhalation
	err = m.SetLEDMode()
	if err != nil {
		log.Panicln(err)
	}

	log.Println("SetDetectionZone")
	err = m.SetDetectionZone(start, end)
	if err != nil {
		log.Panicln(err)
	}

	log.Println("SetSensitivity")
	err = m.SetSensitivity(int(sensivity))
	if err != nil {
		log.Panicln(err)
	}

	err = m.Enable("phase")
	if err != nil {
		log.Panicln(err)
	}

	stream := make(chan interface{})

	log.Println("Opening browser to: ", url)
	open(url)

	go m.Run(stream)

	for {
		select {
		case s := <-stream:
			switch s.(type) {
			case xethru.Respiration:
				resp <- s.(xethru.Respiration)
			case xethru.BaseBandAmpPhase:
				baseband <- s.(xethru.BaseBandAmpPhase)
			case xethru.Sleep:
				sleep <- s.(xethru.Sleep)
			default:
				log.Printf("%#v", s)
			}

		}
	}
}
