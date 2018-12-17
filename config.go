package main

import (
	"encoding/json"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
)

// ReminderConfig ...
type ReminderConfig struct {
	Discord struct {
		Prefix  string `json:"prefix"`
		Token   string `json:"token"`
		OwnerID string `json:"ownerID"`
	} `json:"discord"`
	Destinygg struct {
		Auth string `json:"auth"`
	} `json:"destinygg"`
	Database struct {
		Dialect          string `json:"dialect"`
		ConnectionString string `json:"connectionString"`
	} `json:"database"`
}

// Load loads config file
func (tc *ReminderConfig) Load(file string) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalf("[CONFIG] failed reading file %s: %v", file, err)
	}
	err = json.Unmarshal(b, &tc)
	if err != nil {
		log.Fatalf("[CONFIG] failed unmarshaling file %s: %v", file, err)
	}
}
