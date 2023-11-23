package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {
	logTicker := time.NewTicker(500 * time.Millisecond)
	lg := logrus.New()
	lg.SetOutput(os.Stdout)
	lg.SetLevel(logrus.DebugLevel)
	lg.SetFormatter(&logrus.JSONFormatter{})

	for {
		<-logTicker.C

		switch which := rand.Float64() < 0.2; which {
		case true:
			lg.WithError(fmt.Errorf("something went wrong")).Error("oof")
		case false:
			lg.Debug("nothing to worry about")

		}
	}

}
