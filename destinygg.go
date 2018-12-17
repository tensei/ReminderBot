package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/karrick/tparse"
	log "github.com/sirupsen/logrus"
	"github.com/voloshink/dggchat"
)

// ReminderDgg ...
type ReminderDgg struct {
	conn *dggchat.Session
}

// NewDestinygg start dgg bot
func (rb *ReminderBot) NewDestinygg() {
	// Create a new client
	dgg, err := dggchat.New(rb.Config.Destinygg.Auth)
	if err != nil {
		log.Fatalln(err)
	}

	// Open a connection
	if err = dgg.Open(); err != nil {
		log.Fatalln(err)
	}

	dgg.AddPMHandler(rb.onDggPrivateMessage)
	dgg.AddErrorHandler(onError)

	rb.Dgg.conn = dgg
	log.Info("[MODULE] destinygg bot started")
}

func (rb *ReminderBot) onDggPrivateMessage(m dggchat.PrivateMessage, s *dggchat.Session) {

	if rb.countDggRemindersForUser(m.User.Nick) >= 3 {
		_ = s.SendPrivateMessage(m.User.Nick, "reminder limit reached (3)")
		return
	}

	args := strings.SplitN(m.Message, " ", 2)
	args = removeEmptyItems(args)
	if len(args) < 2 {
		_ = s.SendPrivateMessage(m.User.Nick, "missing args, expected '<time> <message>'")
		return
	}

	timeArg := strings.ToLower(strings.TrimSpace(args[0]))
	messageArg := strings.TrimSpace(args[1])

	now := time.Now().UTC()
	another, err := tparse.AddDuration(now, timeArg)
	if err != nil {
		log.Infof("invalid time entered, %v", err)
		_ = s.SendPrivateMessage(m.User.Nick, fmt.Sprintf("invalid time entered, %v", err))
		return
	}

	if !another.After(now) {
		// silently do nothing?
		log.Infof("time '%s' is in the past", another.String())
		_ = s.SendPrivateMessage(m.User.Nick, fmt.Sprintf("time '%s' is in the past", another.String()))
		return
	}

	// max 3 months
	// make this changeable for each guild?
	maxDuration := now.AddDate(0, 3, 0)
	if another.After(maxDuration) {
		log.Infof("reminder is too far in the future %s that's %v", another.String(), another.Sub(now))
		_ = s.SendPrivateMessage(m.User.Nick, fmt.Sprintf("reminder is too far in the future %s that's %v", another.String(), another.Sub(now)))
		return
	}

	duration := another.Sub(now)

	r := &Reminder{
		ID:            fmt.Sprintf("%d", m.ID),
		UserID:        m.User.Nick,
		Message:       strings.TrimSpace(messageArg),
		Time:          another,
		DirectMessage: true,
		Platform:      "destinygg",
	}

	log.Infof("[DESTINYGG] new reminder from %s in %s", m.User.Nick, duration)

	// adding it to the database reminders
	rb.AddReminder(r)

	// adding it to the in memory reminders
	rb.reMutex.Lock()
	rb.reminders = append(rb.reminders, r)
	rb.reMutex.Unlock()

	msg := fmt.Sprintf("reminding you(%s) in %s, %s", m.User.Nick, duration.String(), another.Format("02 Jan 06 15:04:05 MST"))

	// send it to the user in a pm
	_ = s.SendPrivateMessage(m.User.Nick, msg)

}

func onError(e string, s *dggchat.Session) {
	log.Warnf("error %s\n", e)
}

func (rdgg *ReminderDgg) remind(r *Reminder) {
	_ = rdgg.conn.SendPrivateMessage(r.UserID, fmt.Sprintf("reminder set: %s", r.Time.Format("02 Jan 06 15:04:05 MST")))
	time.Sleep(500 * time.Millisecond)
	_ = rdgg.conn.SendPrivateMessage(r.UserID, r.Message)
}

func (rb *ReminderBot) countDggRemindersForUser(nick string) int {
	rb.reMutex.Lock()
	defer rb.reMutex.Unlock()

	s := 0

	for _, r := range rb.reminders {
		if r.Platform != "destinygg" {
			continue
		}
		if strings.EqualFold(r.UserID, nick) {
			s++
		}
	}

	return s
}
