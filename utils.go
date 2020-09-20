package main

import (
	"encoding/base64"
	"log"
	"math/rand"
	"os"
	"time"
)

const nanosPerMilli = 1e6

var timeLocation *time.Location

// setTimezone set environment variable for IST time
// and initializes timeLocation for the same
func setTimezone() (err error) {
	err = os.Setenv("TZ", "Asia/Kolkata")
	if err != nil {
		log.Fatalf("unable to set location due to %v", err)
	}

	timeLocation, err = time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Fatalf("unable to set location due to %v", err)
	}

	return err
}

// millisToTime converts Unix millis to time.Time.
func millisToTime(t int64) time.Time {
	return time.Unix(0, t*nanosPerMilli)
}

// getRandomString return a random string of 32 length
func getRandomString() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
