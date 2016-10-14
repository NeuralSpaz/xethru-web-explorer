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
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/NeuralSpaz/xethru"
	"github.com/gorilla/websocket"
	"github.com/tarm/serial"
)

func main() {
	log.Println("X2M200 Web Demo")

	commPort := flag.String("com", "/dev/ttyACM0", "the comm port you wish to use")
	baudrate := flag.Uint("baud", 115200, "the baud rate for the comm port you wish to use")
	sensitivity := flag.Uint("sensitivity", 7, "the sensitivity")
	start := flag.Float64("start", 0.5, "start of dectection zone")
	end := flag.Float64("end", 2.1, "end of dectection zone")
	listen := flag.String("listen", "127.0.0.1:2300", "host:port to start webserver")
	// format := flag.String("format", "json", "format for the log files")
	flag.Parse()

	time.Sleep(time.Second * 1)
	baseband := make(chan xethru.BaseBandAmpPhase)
	resp := make(chan xethru.Respiration)
	sleep := make(chan xethru.Sleep)
	go openXethru(*commPort, *baudrate, *sensitivity, *start, *end, baseband, resp, sleep)

	// initize maps of active websocket connections
	baseBandconnections = make(map[*websocket.Conn]bool)
	respirationconnections = make(map[*websocket.Conn]bool)
	sleepconnections = make(map[*websocket.Conn]bool)

	http.HandleFunc("/ws/bb", baseBandwsHandler)
	http.HandleFunc("/ws/resp", respirationwsHandler)
	http.HandleFunc("/ws/sleep", sleepwsHandler)

	http.Handle("/", http.FileServer(http.Dir("./www")))
	// http.HandleFunc("/js/reconnecting-websocket.min.js", websocketReconnectHandler)
	// http.HandleFunc("/", indexHandler)

	// start webserver in the background
	go func() {
		err := http.ListenAndServe(*listen, nil)
		if err != nil {
			log.Panic(err)
		}
	}()

	// open default browser
	// open("http://" + *listen)

	// Send all the websocket streams as soon as they arrive
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
		case data := <-sleep:
			b, err := json.Marshal(data)
			if err != nil {
				log.Panicln("Error Marshaling: ", err)
			}
			go sendsleep(b)
		}
	}
}

// respirationfile, err := os.Create("./respiration.json")
// if err != nil {
// 	log.Panic(err)
// }
// defer respirationfile.Close()
//
// sleepfile, err := os.Create("./sleep.json")
// if err != nil {
// 	log.Panic(err)
// }
// defer sleepfile.Close()
//
// sleepenc := json.NewEncoder(sleepfile)
// respirationenc := json.NewEncoder(respirationfile)
// s = s.(xethru.Sleep)
// if err := sleepenc.Encode(&s); err != nil {
// 	log.Println(err)
// }
// if err := respirationenc.Encode(&s); err != nil {
// 	log.Println(err)
// }

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

func reset(comm string, baudrate uint) error {
	c := &serial.Config{Name: comm, Baud: int(baudrate)}

	port, err := serial.OpenPort(c)
	if err != nil {
		log.Printf("serial.Open: %v\n", err)
	}
	port.Flush()

	x2 := xethru.Open("x2m200", port)
	// defer port.Close()
	defer x2.Close()

	reset, err := x2.Reset()
	if err != nil {
		log.Printf("serial.Reset: %v\n", err)
		return err
	}
	if !reset {
		log.Panic("Could not reset")
	}
	return nil
}

func openXethru(comm string, baudrate uint, sensivity uint, start float64, end float64, baseband chan xethru.BaseBandAmpPhase, resp chan xethru.Respiration, sleep chan xethru.Sleep) {

	err := reset(comm, baudrate)
	if err != nil {
		log.Panic(err)
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

	c := &serial.Config{Name: comm, Baud: int(baudrate)}
	port, err := serial.OpenPort(c)
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
