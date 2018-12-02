package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type commandFunc func(s *discordgo.Session, m *discordgo.MessageCreate, command string, dm bool)

var (
	invalidTimeEmbed = &discordgo.MessageEmbed{
		Description: "invalid time, please use 1m (or w|d|h|m) time format",
		Color:       0xFF0000,
	}
	maxPublicReminders    = 3
	maxPublicReachedEmbed = &discordgo.MessageEmbed{
		Description: "only 3 public reminders allowed, please use dm's to add more",
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
		prefix + "uptime":      discordUptime(rb),
		prefix + "stats":       discordStats(rb),
		prefix + "remind":      discordRemind(rb),
		prefix + "getreminder": discordGetReminder(rb),
		prefix + "help":        discordHelp(rb),
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

func discordGetReminder(rb *ReminderBot) func(s *discordgo.Session, m *discordgo.MessageCreate, command string, dm bool) {
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
				&discordgo.MessageEmbedField{
					Name:   "started",
					Value:  rb.started.UTC().Format(time.RFC822),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
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
				&discordgo.MessageEmbedField{
					Name:   "guilds",
					Value:  fmt.Sprintf("%d", guilds),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "users",
					Value:  fmt.Sprintf("%d", users),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "reminders",
					Value:  fmt.Sprintf("%d", len(rb.reminders)),
					Inline: true,
				},
			},
		})
	}
}

var (
	timeFormatRegex = regexp.MustCompile("^[0-9]+[wdhm]")
	timeRegex       = regexp.MustCompile("^([0-9]+w)?([0-9]+d)?([0-9]+h)?([0-9]+m)?$")
)

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
		if len(args) < 3 {
			return
		}
		args = args[1:]

		timeArg := strings.ToLower(strings.TrimSpace(args[0]))
		messageArg := strings.TrimSpace(args[1])

		if !timeFormatRegex.MatchString(timeArg) {
			s.ChannelMessageSendEmbed(m.ChannelID, invalidTimeEmbed)
			return
		}
		matches := timeRegex.FindAllStringSubmatch(timeArg, -1)
		if len(matches) < 1 {
			s.ChannelMessageSendEmbed(m.ChannelID, invalidTimeEmbed)
			return
		}
		if len(matches[0]) < 1 {
			s.ChannelMessageSendEmbed(m.ChannelID, invalidTimeEmbed)
			return
		}

		var duration time.Duration
		for _, m := range matches[0][1:] {
			if m == "" {
				continue
			}
			suffix := m[len(m)-1:]
			switch suffix {
			case "w":
				w, _ := strconv.Atoi(strings.TrimSuffix(m, "w"))
				duration += time.Duration(w) * ((time.Hour * 24) * 7)
			case "d":
				d, _ := strconv.Atoi(strings.TrimSuffix(m, "d"))
				duration += time.Duration(d) * (time.Hour * 24)
			case "h":
				h, _ := strconv.Atoi(strings.TrimSuffix(m, "h"))
				duration += time.Duration(h) * time.Hour
			case "m":
				m, _ := strconv.Atoi(strings.TrimSuffix(m, "m"))
				duration += time.Duration(m) * time.Minute
			}
		}
		// log.Infof("Adding reminder for %s, %s", duration, messageArg)
		r := &Reminder{
			ID:            m.ID,
			UserID:        m.Author.ID,
			ChannelID:     m.ChannelID,
			Message:       strings.TrimSpace(messageArg),
			Time:          time.Now().UTC().Add(duration),
			DirectMessage: dm,
		}

		// don't need to set this if it isn't a dm
		if !dm {
			r.Mentions = allMentions
		}

		// adding it to the database reminders
		rb.AddReminder(r)
		rb.reMutex.Lock()
		// adding it to the in memory reminders
		rb.reminders = append(rb.reminders, r)
		rb.reMutex.Unlock()

		msg := fmt.Sprintf("reminding you(%s) in %s, %s", m.Author.Mention(), duration, time.Now().UTC().Add(duration).Format("02 Jan 06 15:04:05 MST"))

		if dm {
			// create dm chat with the user
			dmCh, err := s.UserChannelCreate(m.Author.ID)
			if err != nil {
				log.Warnf("couldn't create dm channel, %v", err)
				return
			}
			s.ChannelMessageSendEmbed(dmCh.ID, &discordgo.MessageEmbed{
				Description: msg,
			})
			return
		}

		// send it to the public #channel
		s.ChannelMessageSendEmbed(r.ChannelID, &discordgo.MessageEmbed{
			Description: msg,
		})
	}
}

func (rd *ReminderDiscord) remind(r *Reminder) {
	embed := &discordgo.MessageEmbed{
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "set",
				Value:  r.CreatedAt.UTC().Format("02 Jan 06 15:04:05 MST"),
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "user",
				Value:  fmt.Sprintf("<@%s>", r.UserID),
				Inline: true,
			},
		},
		Description: r.Message,
	}
	if r.DirectMessage {
		dm, err := rd.c.UserChannelCreate(r.UserID)
		if err != nil {
			log.Warnf("couldn't create dm channel, %v", err)
			return
		}
		rd.c.ChannelMessageSendEmbed(dm.ID, embed)
		return
	}
	rd.c.ChannelMessageSendEmbed(r.ChannelID, embed)
	rd.c.ChannelMessageSend(r.ChannelID, r.Mentions)
}

func (rb *ReminderBot) countPublicRemindersUser(userID string) int {
	c := 0
	rb.reMutex.Lock()
	defer rb.reMutex.Unlock()

	for _, r := range rb.reminders {
		if !r.DirectMessage && r.UserID == userID {
			c++
		}
	}
	return c
}
