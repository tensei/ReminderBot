package main

import (
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	log "github.com/sirupsen/logrus"
)

// Guild struct for database
type Guild struct {
	ID        string `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time

	OwnerName string
	OwnerID   string
}

// Reminder ...
type Reminder struct {
	ID        string `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time

	UserID        string
	ChannelID     string
	Message       string
	Time          time.Time
	DirectMessage bool
	Mentions      string
	Platform      string
}

// NewDatabase create/opens a database
func (rb *ReminderBot) NewDatabase() {
	db := rb.Config.Database.ConnectionString
	dialect := rb.Config.Database.Dialect

	if db == "" || dialect == "" {
		log.Fatal("[DATABASE] missing connectionString/dialect in config file")
	}

	var err error
	rb.db, err = gorm.Open(dialect, db)
	if err != nil {
		log.Fatalf("[DATABASE] failed opening %s: %v", db, err)
	}

	rb.db.AutoMigrate(&Guild{})
	rb.db.AutoMigrate(&Reminder{})

	log.Info("[MODULE] database loaded")
}

// AddGuildToDB add new guild to database
func (rb *ReminderBot) AddGuildToDB(g *discordgo.Guild, owner *discordgo.User) {
	var dbg Guild
	rb.db.Where("id = ?", g.ID).First(&dbg)
	if dbg.ID == g.ID {
		return
	}
	dbg.ID = g.ID
	dbg.OwnerID = owner.ID
	dbg.OwnerName = owner.String()
	rb.db.Create(&dbg)
}

// GetAllReminders return all reminders in the database
func (rb *ReminderBot) GetAllReminders() []*Reminder {
	var reminders []*Reminder
	rb.db.Find(&reminders)
	return reminders
}

// AddReminder adds reminder to database
func (rb *ReminderBot) AddReminder(r *Reminder) {
	rb.db.Create(r)
}

// RemoveReminder adds reminder to database
func (rb *ReminderBot) RemoveReminder(r *Reminder) {
	rb.db.Delete(r)
}
