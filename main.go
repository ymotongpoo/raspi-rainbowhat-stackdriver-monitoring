// Copyright 2019 Yoshi Yamaguchi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/devices/apa102"
	"periph.io/x/periph/devices/bmxx80"
	"periph.io/x/periph/experimental/devices/rainbowhat"
	"periph.io/x/periph/host"
)

const Interval = 5 * time.Second

type BMP280Data struct {
	Temperature float64
	Pressure    float64
}

func main() {
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	hat, err := rainbowhat.NewRainbowHat(&apa102.DefaultOpts)
	if err != nil {
		log.Fatalf("failed to initialize Rainbow HAT: %v", err)
	}
	defer hat.Halt()

	//sigs := make(chan os.Signal)
	//signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	sensor := hat.GetBmp280()
	ticker := time.NewTicker(Interval)
	defer ticker.Stop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(sig)

	dataCh := make(chan BMP280Data, 10)
	for {
		select {
		case <-ticker.C:
			d, err := getSensorData(sensor)
			if err != nil {
				log.Printf("error on sensing BMP280: %v\n", err)
				break
			}
			dataCh <- d
		case d := <-dataCh:
			fmt.Printf("Temp: %v, Pressure: %v\n", d.Temperature, d.Pressure)
		case s := <-sig:
			fmt.Println(s)
			return
		}
	}
}

func getSensorData(s *bmxx80.Dev) (BMP280Data, error) {
	var env physic.Env
	if err := s.Sense(&env); err != nil {
		return BMP280Data{}, err
	}
	d := BMP280Data{
		Temperature: float64(env.Temperature-physic.ZeroCelsius) / float64(physic.Kelvin),
		Pressure:    float64(env.Pressure) / float64(100*physic.Pascal),
	}
	return d, nil
}
