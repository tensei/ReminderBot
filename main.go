package main

import (
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

// ReminderBot ...
type ReminderBot struct {
	db      *gorm.DB
	Discord *ReminderDiscord
	Config  *ReminderConfig

	reMutex   sync.Mutex
	reminders []*Reminder

	started time.Time
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

func main() {
	rb := NewReminderBot()

	defer rb.Close()
	rb.Config.Load("config.json")

	rb.NewDatabase()
	rb.NewDiscord()

	// load database reminders after start
	rb.reMutex.Lock()
	rb.reminders = append(rb.reminders, rb.GetAllReminders()...)
	rb.reMutex.Unlock()
	log.Infof("Loaded %d reminder(s)", len(rb.reminders))
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
		Discord:   new(ReminderDiscord),
		Config:    new(ReminderConfig),
		reminders: []*Reminder{},
		started:   time.Now().UTC(),
	}
}

// Close closing everything
func (rb *ReminderBot) Close() {
	_ = rb.db.Close()
	_ = rb.Discord.c.Close()
}

func (rb *ReminderBot) startReminding() {
	tick := time.NewTicker(time.Minute)
	for range tick.C {
		rb.reMutex.Lock()
		var toKeep []*Reminder
		for _, r := range rb.reminders {
			if r == nil {
				continue
			}
			if time.Now().UTC().After(r.Time) {

				// create a dm channel and remind him/her/it
				rb.Discord.remind(r)

				// database too
				rb.RemoveReminder(r)
			} else {
				// keep this bitch
				toKeep = append(toKeep, r)
			}
		}
		rb.reminders = toKeep

		rb.reMutex.Unlock()
	}
}
