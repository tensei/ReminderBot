package main

import (
	"os"
	"os/signal"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

// ReminderBot ...
type ReminderBot struct {
	db      *gorm.DB
	Discord *ReminderDiscord
	Dgg     *ReminderDgg
	Config  *ReminderConfig

	started time.Time
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

func main() {
	rb := NewReminderBot()

	defer rb.Close()
	rb.Config.Load("config.json")

	rb.NewDatabase()
	rb.NewDiscord()
	rb.NewDestinygg()

	log.Infof("Loaded %d reminder(s)", len(rb.GetAllReminders()))
	go rb.startReminding()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c
}

// NewReminderBot ...
func NewReminderBot() *ReminderBot {
	return &ReminderBot{
		Discord: new(ReminderDiscord),
		Dgg:     new(ReminderDgg),
		Config:  new(ReminderConfig),
		started: time.Now().UTC(),
	}
}

// Close closing everything
func (rb *ReminderBot) Close() {
	_ = rb.db.Close()
	_ = rb.Discord.c.Close()
	_ = rb.Dgg.conn.Close()
}

func (rb *ReminderBot) startReminding() {
	tick := time.NewTicker(time.Minute)
	for range tick.C {
		reminders := rb.GetAllReminders()
		for _, r := range reminders {
			if r == nil {
				continue
			}
			if time.Now().UTC().After(r.Time) {

				switch r.Platform {
				case "discord", "":
					// create a dm channel if dm and remind him/her/it
					rb.Discord.remind(r)
				case "destinygg":
					// send reminder as pm to dgg user
					rb.Dgg.remind(r)
				}

				// database too
				rb.RemoveReminder(r)
			}
		}
	}
}
