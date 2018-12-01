package main

import (
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// GuildCreate handles guild join events
func (rb *ReminderBot) GuildCreate(s *discordgo.Session, m *discordgo.GuildCreate) {
	// add guild to db
	owner, err := s.User(m.OwnerID)
	if err != nil {
		log.Warnf("[JOIN] error getting owner information, %v", err)
		return
	}
	log.Infof("[JOIN] guild: %s(%s), owner: %s(%s), member_count: %d", m.Name, m.ID, owner.String(), m.OwnerID, m.MemberCount)
	rb.AddGuildToDB(m.Guild, owner)
}
