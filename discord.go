package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/karrick/tparse"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type commandFunc func(s *discordgo.Session, m *discordgo.MessageCreate, command string, dm bool)

var (
	invalidTimeEmbed = &discordgo.MessageEmbed{
		Description: "invalid time, please use 1m, 1.5h or 2h45m (y|mo|w|d|h|m) time format",
		Color:       0xFF0000,
	}
	missingArgsEmbed = &discordgo.MessageEmbed{
		Description: "missing args, please use !remind <duration> <text>",
		Color:       0xFF0000,
	}
	pastTimeErrEmbed = &discordgo.MessageEmbed{
		Description: "can't remind you in the past...",
		Color:       0xFF0000,
	}

	maxPublicReminders    = 5
	maxPublicReachedEmbed = &discordgo.MessageEmbed{
		Description: "only 3 public reminders allowed, please use dm's to add more",
		Color:       0xFF0000,
	}
	maxPrivateReminders    = 10
	maxPrivateReachedEmbed = &discordgo.MessageEmbed{
		Description: "only 10 private reminders allowed",
		Color:       0xFF0000,
	}

	maxDurationReachedEmbed = &discordgo.MessageEmbed{
		Description: "max. 3 months please, you are over that",
		Color:       0xFF0000,
	}
)

// ReminderDiscord discord part of the bot
type ReminderDiscord struct {
	c      *discordgo.Session
	prefix string

	commands map[string]commandFunc
}

// SetupDiscordCommands ...
func (rb *ReminderBot) SetupDiscordCommands() {
	prefix := rb.Config.Discord.Prefix
	rb.Discord.commands = map[string]commandFunc{
		prefix + "uptime":       discordUptime(rb),
		prefix + "stats":        discordStats(rb),
		prefix + "remind":       discordRemind(rb),
		prefix + "getreminder":  discordGetReminder(),
		prefix + "helpreminder": discordHelp(rb),
	}
}

// NewDiscord creates a new discord session
func (rb *ReminderBot) NewDiscord() {
	token := rb.Config.Discord.Token
	if token == "" {
		log.Fatal("[DISCORD] missing TOKEN in config file")
	}

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("[DISCORD] failed creating new session: %v", err)
	}

	s.AddHandler(rb.CommandHandler)
	s.AddHandler(rb.GuildCreate)

	rb.SetupDiscordCommands()

	err = s.Open()
	if err != nil {
		log.Fatalf("failed opening connection to discord: %v", err)
	}
	_ = s.UpdateStatus(0, fmt.Sprintf("%shelpreminder", rb.Config.Discord.Prefix))

	rb.Discord.c = s
	rb.Discord.prefix = rb.Config.Discord.Prefix
	log.Info("[MODULE] discord loaded")
}

// CommandHandler ...
func (rb *ReminderBot) CommandHandler(s *discordgo.Session, m *discordgo.MessageCreate) {

	if !strings.HasPrefix(m.Content, rb.Discord.prefix) {
		return
	}

	parts := strings.SplitN(m.Content, " ", 2)
	for k, fn := range rb.Discord.commands {
		if strings.EqualFold(k, parts[0]) {
			dm := false
			guild, err := s.Guild(m.GuildID)
			if err != nil {
				log.Infof("[COMMAND] %s used in dm user: %s(%s)", parts[0], m.Author.String(), m.Author.ID)
				dm = true
			} else {
				log.Infof("[COMMAND] %s used in server: %s(%s), user: %s(%s)", parts[0], guild.Name, guild.ID, m.Author.String(), m.Author.ID)
			}
			go fn(s, m, k, dm)
			return
		}
	}
}

func (rb *ReminderBot) isOwner(id string) bool {
	return rb.Config.Discord.OwnerID == id
}

func discordGetReminder() func(s *discordgo.Session, m *discordgo.MessageCreate, command string, dm bool) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate, command string, dm bool) {
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Description: "https://discordapp.com/oauth2/authorize?client_id=517763064351686656&scope=bot&permissions=1",
		})
	}
}

func discordHelp(rb *ReminderBot) func(s *discordgo.Session, m *discordgo.MessageCreate, command string, dm bool) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate, command string, dm bool) {
		prefix := rb.Config.Discord.Prefix
		msg := fmt.Sprintf("%sremind 1h (w for weeks, d for Days, h for Hours, m for Minutes) your text here\n", prefix)
		msg += fmt.Sprintf("%sgetreminder returns the link to add the bot to your server", prefix)

		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Description: msg,
		})
	}
}

func discordUptime(rb *ReminderBot) func(s *discordgo.Session, m *discordgo.MessageCreate, command string, dm bool) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate, command string, dm bool) {
		if !rb.isOwner(m.Author.ID) {
			return
		}
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "started",
					Value:  rb.started.UTC().Format(time.RFC822),
					Inline: true,
				},
				{
					Name:   "uptime",
					Value:  fmt.Sprintf("%s", time.Since(rb.started)),
					Inline: true,
				},
			},
		})
	}
}

func discordStats(rb *ReminderBot) func(s *discordgo.Session, m *discordgo.MessageCreate, command string, dm bool) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate, command string, dm bool) {
		if !rb.isOwner(m.Author.ID) {
			return
		}
		guilds := len(s.State.Guilds)
		users := 0
		for _, g := range s.State.Guilds {
			users += g.MemberCount
		}

		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "guilds",
					Value:  fmt.Sprintf("%d", guilds),
					Inline: true,
				},
				{
					Name:   "users",
					Value:  fmt.Sprintf("%d", users),
					Inline: true,
				},
				{
					Name:   "reminders",
					Value:  fmt.Sprintf("%d", len(rb.GetAllReminders())),
					Inline: true,
				},
			},
		})
	}
}

func discordRemind(rb *ReminderBot) func(s *discordgo.Session, m *discordgo.MessageCreate, command string, dm bool) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate, command string, dm bool) {
		// Ignore all messages created by the bot itself
		if m.Author.ID == s.State.User.ID {
			return
		}

		if !dm && rb.countPublicRemindersUser(m.Author.ID) >= maxPublicReminders {
			s.ChannelMessageSendEmbed(m.ChannelID, maxPublicReachedEmbed)
			return
		}
		if dm && rb.countPrivateRemindersUser(m.Author.ID) >= maxPrivateReminders {
			s.ChannelMessageSendEmbed(m.ChannelID, maxPrivateReachedEmbed)
			return
		}

		content := m.Content
		allMentions := ""
		if !dm {
			// fill mentions string
			allMentions = m.Author.Mention()
			// user mentions
			for _, u := range m.Mentions {
				allMentions += fmt.Sprintf(" %s", u.Mention())
			}

			// role mentions
			for _, role := range m.MentionRoles {
				allMentions += fmt.Sprintf(" <@&%s>", role)
			}

			// replace mentions
			co, err := m.ContentWithMoreMentionsReplaced(s)
			if err != nil {
				co = m.ContentWithMentionsReplaced()
			}
			content = co
		}

		args := strings.SplitN(content, " ", 3)
		args = removeEmptyItems(args)
		if len(args) < 3 {
			s.ChannelMessageSendEmbed(m.ChannelID, missingArgsEmbed)
			return
		}
		args = args[1:]

		timeArg := strings.ToLower(strings.TrimSpace(args[0]))
		messageArg := strings.TrimSpace(args[1])

		now := time.Now().UTC()
		another, err := tparse.AddDuration(now, timeArg)
		if err != nil {
			log.Infof("invalid time entered, %v", err)
			s.ChannelMessageSendEmbed(m.ChannelID, invalidTimeEmbed)
			return
		}

		if !another.After(now) {
			// silently do nothing?
			log.Infof("time '%s' is in the past", another.String())
			s.ChannelMessageSendEmbed(m.ChannelID, pastTimeErrEmbed)
			return
		}

		// max 3 months
		// make this changeable for each guild?
		maxDuration := now.AddDate(0, 3, 0)
		if another.After(maxDuration) {
			log.Infof("reminder is too far in the future %s that's %v", another.String(), another.Sub(now))
			s.ChannelMessageSendEmbed(m.ChannelID, maxDurationReachedEmbed)
			return
		}

		duration := another.Sub(now)

		r := &Reminder{
			ID:            m.ID,
			UserID:        m.Author.ID,
			ChannelID:     m.ChannelID,
			GuildID:       m.GuildID,
			Message:       strings.TrimSpace(messageArg),
			Time:          another,
			DirectMessage: dm,
			Platform:      "discord",
		}

		// don't need to set this if it isn't a dm
		if !dm {
			r.Mentions = allMentions
		}

		// adding it to the database reminders
		rb.AddReminder(r)

		msg := fmt.Sprintf("reminding you(%s) in %s, %s", m.Author.Mention(), duration.String(), another.Format("02 Jan 06 15:04:05 MST"))

		// send it to the public #channel
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Description: msg,
		})
	}
}

func (rd *ReminderDiscord) remind(r *Reminder) {
	m := &discordgo.MessageSend{
		Content: r.Mentions,
		Embed: &discordgo.MessageEmbed{
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "set",
					Value:  r.CreatedAt.UTC().Format("02 Jan 06 15:04:05 MST"),
					Inline: true,
				},
				{
					Name:   "user",
					Value:  fmt.Sprintf("<@%s>", r.UserID),
					Inline: true,
				},
			},
			Description: r.Message,
		},
	}
	chID := r.ChannelID
	if r.DirectMessage {
		dm, err := rd.c.UserChannelCreate(r.UserID)
		if err != nil {
			log.Warnf("couldn't create dm channel, %v", err)
			return
		}
		chID = dm.ID
	}
	rd.c.ChannelMessageSendComplex(chID, m)
}

func (rb *ReminderBot) countPublicRemindersUser(userID string) int {
	c := 0
	reminders := rb.GetAllReminders()

	for _, r := range reminders {
		if !r.DirectMessage && r.UserID == userID {
			c++
		}
	}
	return c
}

func (rb *ReminderBot) countPrivateRemindersUser(userID string) int {
	c := 0
	reminders := rb.GetAllReminders()

	for _, r := range reminders {
		if r.DirectMessage && r.UserID == userID {
			c++
		}
	}
	return c
}

func removeEmptyItems(items []string) []string {
	var n []string
	for _, i := range items {
		if !strings.EqualFold(strings.TrimSpace(i), "") {
			n = append(n, i)
		}
	}
	return n
}
