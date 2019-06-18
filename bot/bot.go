package bot

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"../config"
	"github.com/Necroforger/dgrouter/exrouter"
	"github.com/bwmarrin/discordgo"
)

var (
	BotID string
	Bot   *discordgo.User
	goBat *discordgo.Session
	e, e2 discordgo.Emoji

	playerInfo         []playerInfoStruct
	suggestion         []suggestionStruct
	battle             []battleStruct
	servant            []servantStruct
	blankInfo          playerInfoStruct
	channels           *discordgo.Channel
	router             = exrouter.New()
	suggestionChannel  string
	suggestionNum      int
	suggestionUser     int
	suggestionmsgCheck bool
	suggestionID       string
	Running            int
	Line               string
	Original           string
	once               bool
	sendOnce           bool
	suggestionListener bool
	checker            bool
	solved             bool
	timeInterval       time.Time
	timeSolve          time.Time
	miniGameStart      bool
	channelsID         string
	pvpchannel         string
	timeNow            time.Time = time.Now().In(time.FixedZone("UTC+9", 9*60*60))
	resetTime          time.Time = time.Date(timeNow.Year(), timeNow.Month(), timeNow.Day(), 24, 00, 00, 00, time.FixedZone("UTC+9", 9*60*60))
)

type suggestionStruct struct {
	Username   string
	Server     []string
	UserID     string
	Suggestion []string
}

type servantStruct struct {
	Name     string
	UserID   string
	Servants []string
}

type configStruct struct {
	Token     string `json:"Token"`
	BotPrefix string `json:"BotPrefix"`
}

type playerInfoStruct struct {
	Name                string
	UserID              string
	SaintQuartz         int
	SaintQuartzFragment int
	BonusDay            int
	Booster             int
	BoosterUsed         bool
	DailyCollect        bool
	LastCollectedNitro  time.Time
	UsedBooster         time.Time
}

type battleStruct struct {
	UserID             string    // Personal ID
	Username           string    // Requester's ID
	RequesterName      string    // Requester's Name
	RequesterID        string    // Requester's ID
	RequestedName      string    // Requested's Name
	RequestedID        string    // Requested's ID
	RequestedForBattle bool      // Check if the player is already requested for a battle
	InABattle          bool      // Check if the player is in a battle
	RequestedSomeone   bool      // Checks if the player requested a battle with someone
	RequestedTime      time.Time // Time when they requested a User
}

func Start() {
	gotBot, err := discordgo.New("Bot " + config.Token)
	goBat = gotBot
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	u, err := gotBot.User("@me")

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	Bot = u
	BotID = u.ID
	e.Name = "IconSQ"
	e.ID = "378555954305826818"
	e2.Name = "IconSQFragment"
	e2.ID = "587614978052325376"

	err = gotBot.Open()
	fmt.Println("Bot is running!")

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	gotBot.AddHandler(messageHandler)
	gotBot.AddHandler(listen)
	gotBot.AddHandler(boosterExpiry)

	d := discordgo.UpdateStatusData{
		Game: &discordgo.Game{
			Type: discordgo.GameTypeStreaming,
			Name: "with Tsubasa!",
			URL:  "https://www.twitch.tv/amamitsubasaaa",
		},
	}

	if err := gotBot.UpdateStatusComplex(d); err != nil {
		fmt.Println(err.Error())
		return
	}

	err = gotBot.UpdateStreamingStatus(0, "with Tsubasa!", "https://www.twitch.tv/amamitsubasaaa")

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	servers := goBat.State.Guilds
	fmt.Println("ScathachBot is running on ", len(servers), " servers!")

	for _, guild := range goBat.State.Guilds {
		channels, _ := goBat.GuildChannels(guild.ID)
		for _, channel := range channels {
			if channel.Name == "scathach-bot-minigame" {
				miniGameStart = true
				channelsID = channel.ID
				break
			}
		}
		break
	}

	go func() {
		for {
			if miniGameStart {
				for range time.NewTicker(time.Second).C {
					miniGame(gotBot, channelsID)
					if !miniGameStart {
						break
					}
				}
			} else {
				if solved {
					for range time.NewTicker(time.Minute).C {
						miniGameStart = true
						break
					}
				} else {
					for range time.NewTicker(5 * time.Minute).C {
						miniGameStart = true
						break
					}
				}

			}
		}
	}()

	for _, guild := range goBat.State.Guilds {
		channels, _ := goBat.GuildChannels(guild.ID)
		for _, channel := range channels {
			if channel.Name == "scathach-bot-pvp" {
				pvpchannel = channel.ID
				break
			}
		}
		break
	}

	go func() {
		for range time.NewTicker(time.Second).C {
			checkPvp(gotBot, pvpchannel)
		}
	}()
}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	commands(s, m)
	router.FindAndExecute(s, config.BotPrefix, BotID, m.Message)
}

func listen(s *discordgo.Session, m *discordgo.MessageCreate) {
	rand.Seed(time.Now().UnixNano())

	if m.Author.ID == BotID {
		return
	}

	var channelID string
	var message bool
	Guild, _ := s.Guild(m.GuildID)
	channel := Guild.Channels
	for channels := range Guild.Channels {
		if channel[channels].Name == "scathach-bot-minigame" {
			channelID = channel[channels].ID
			message = true
			break
		}
	}

	if message == false {
		fmt.Println("Can't find any channel named scathach-bot-minigame")
	}

	if checker && m.Author.ID == suggestionID {
		if strings.ToLower(m.Content) == "yes" {
			suggestionListener = true
			suggestionmsgCheck = true
		} else if strings.ToLower(m.Content) == "no" {
			suggestionListener = false
			suggestionmsgCheck = true
		} else {
			embed := discordgo.MessageEmbed{
				Description: "Unknown reply. Reply with Yes/No",
				Color:       8782097,
			}
			_, _ = s.ChannelMessageSendEmbed(suggestionChannel, &embed)
		}

		if checker && suggestionmsgCheck {
			if suggestionListener {
				embed := discordgo.MessageEmbed{
					Description: "Successfully deleted:\n ```diff\n- " + suggestion[suggestionUser].Suggestion[suggestionNum] + "```",
					Color:       32768,
				}
				copy(suggestion[suggestionUser].Suggestion[suggestionNum:], suggestion[suggestionUser].Suggestion[suggestionNum+1:])
				suggestion[suggestionUser].Suggestion[len(suggestion[suggestionUser].Suggestion)-1] = ""
				suggestion[suggestionUser].Suggestion = suggestion[suggestionUser].Suggestion[:len(suggestion[suggestionUser].Suggestion)-1]
				_, _ = s.ChannelMessageSendEmbed(suggestionChannel, &embed)
				SaveSuggestion()
				suggestionListener = false
				checker = false
				return
			} else {
				embed := discordgo.MessageEmbed{
					Description: "Successfully cancelled",
					Color:       32768,
				}
				_, _ = s.ChannelMessageSendEmbed(suggestionChannel, &embed)
				suggestionListener = false
				checker = false
				return
			}
		}
	}

	if strings.ToLower(m.Content) == strings.ToLower(Original) && m.Message.ChannelID == channelID {
		if Running == 1 {
			Running = 0
			for i := 0; i < len(playerInfo); i++ {
				if playerInfo[i].UserID == m.Message.Author.ID {
					if playerInfo[i].BoosterUsed {
						playerInfo[i].SaintQuartzFragment += 2
						embed := discordgo.MessageEmbed{
							Title:       "Boosted",
							Description: m.Author.Mention() + " solved the scrambled word, **" + Original + "** and received 2x the reward",
							Color:       16023551,
						}
						_, _ = s.ChannelMessageSendEmbed(channelID, &embed)
						SavePlayerInfo()
						once = false
						sendOnce = false
						miniGameStart = false
						solved = true
						return
					} else {
						playerInfo[i].SaintQuartzFragment += 1
						embed := discordgo.MessageEmbed{
							Description: m.Author.Mention() + " solved the scrambled word, **" + Original + "** and received the reward",
							Color:       11454159,
						}
						_, _ = s.ChannelMessageSendEmbed(channelID, &embed)
						SavePlayerInfo()
						once = false
						sendOnce = false
						miniGameStart = false
						solved = true
						return
					}
				}
			}
			fmt.Println("Creating New User...")
			CreateNewUser(m.Author.ID, m.Author.Username)
			for i := 0; i < len(playerInfo); i++ {
				if playerInfo[i].UserID == m.Message.Author.ID {
					if playerInfo[i].BoosterUsed {
						playerInfo[i].SaintQuartzFragment += 2
						embed := discordgo.MessageEmbed{
							Title:       "Boosted",
							Description: m.Author.Mention() + " solved the scrambled word, **" + Original + "** and received 2x the reward",
							Color:       11454159,
						}
						_, _ = s.ChannelMessageSendEmbed(channelID, &embed)
						SavePlayerInfo()
						once = false
						sendOnce = false
						miniGameStart = false
						solved = true
						return
					} else {
						playerInfo[i].SaintQuartzFragment += 1
						embed := discordgo.MessageEmbed{
							Description: m.Author.Mention() + " solved the scrambled word, **" + Original + "** and received the reward",
							Color:       11454159,
						}
						_, _ = s.ChannelMessageSendEmbed(channelID, &embed)
						SavePlayerInfo()
						once = false
						sendOnce = false
						miniGameStart = false
						solved = true
						return
					}
				}
			}
		}
	}

	if len(m.Mentions) > 0 && m.Mentions[0].ID == BotID {
		matched, err := regexp.Match(`^(Hello|hello|Hi|hi|Hey|hey) .*$`, []byte(m.Content))

		if err != nil {
			fmt.Println(err.Error())
			return
		}

		if matched {
			var RandNum int
			RandNum = rand.Intn(4)

			if RandNum == 1 {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Hello there, "+m.Author.Mention()+"! :>")
			} else if RandNum == 2 {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Hey, "+m.Author.Mention()+", How are you today?")
			} else if RandNum == 3 {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Hello, "+m.Author.Mention()+"! Nice to meet you!")
			} else if RandNum == 4 {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Oh Hi, "+m.Author.Mention()+"! Have a good day!")
			} else {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Hi Hi, "+m.Author.Mention()+"! What are you up to today?")
			}
			matched = false
		}
	}
}

func commands(s *discordgo.Session, m *discordgo.MessageCreate) {
	router.Group(func(r *exrouter.Route) {
		r.Cat("Admin")
		r.Use(Auth)
		r.On("addbalance", func(ctx *exrouter.Context) { addBalance(ctx) }).Desc("Add new balance to user")
		r.On("resetbalance", func(ctx *exrouter.Context) { resetBalance(ctx) }).Desc("Resets user's balance to 0")
		r.On("buyrole", func(ctx *exrouter.Context) { buyRole(s, ctx) }).Desc("Buys a role for 500 SQs. Syntax: " + config.BotPrefix + "buyrole [RoleName] [Color]")
		r.On("prefix", func(ctx *exrouter.Context) { prefix(ctx) }).Desc("Change the bot's prefix")
	})
	router.On("ping", func(ctx *exrouter.Context) { ctx.Reply("pong") }).Desc("Responds with pong")
	router.On("daily", func(ctx *exrouter.Context) { daily(ctx) }).Desc("Claim your daily gift")
	router.On("shinda", func(ctx *exrouter.Context) { ctx.Reply("You're not human! " + ctx.Msg.Author.Mention()) }).Desc("Responds with You're not human! and pings yourself")
	router.On("balance", func(ctx *exrouter.Context) { balance(ctx) }).Desc("Checks balance. Syntax: " + config.BotPrefix + "balance/ " + config.BotPrefix + "balance [User]")
	router.On("bal", func(ctx *exrouter.Context) { balance(ctx) })
	router.On("dice", func(ctx *exrouter.Context) { dice(ctx) }).Desc("Returns a random value between 1-6")
	router.On("exchangesqf", func(ctx *exrouter.Context) { exchangeSQF(ctx) }).Desc("Exchanges all your Saint Quartz Fragments into Saint Quartz. 7SQ Fragment = 1SQ")
	router.On("shop", func(ctx *exrouter.Context) { shop(ctx) }).Desc("Show items in the shop")
	router.On("suggest", func(ctx *exrouter.Context) { suggest(ctx) }).Desc("Check your suggestions/Suggest ideas for the bot. Syntax:" + config.BotPrefix + "suggest / " + config.BotPrefix + "suggest [Suggestion]")
	router.On("rsuggest", func(ctx *exrouter.Context) { askSuggest(ctx) }).Desc("Removes a suggestion Syntax:" + config.BotPrefix + "rsuggest [SuggestionNumber]")
	router.On("nclaim", func(ctx *exrouter.Context) { nitroclaim(ctx) })
	router.On("nitroclaim", func(ctx *exrouter.Context) { nitroclaim(ctx) }).Desc("Only for Nitro Boosters. Claims a booster for minigame that x2 the reward given")
	router.On("boost", func(ctx *exrouter.Context) { useBooster(ctx) }).Desc("Only for Nitro Boosters. Uses your booster for minigame")
	router.On("battleinfo", func(ctx *exrouter.Context) { infoBattle(ctx) }).Desc("Check out what battle does")
	router.On("battle", func(ctx *exrouter.Context) { reqBattle(ctx) }).Desc("Request battle with a user")
	router.On("accept", func(ctx *exrouter.Context) { acceptBattle(ctx) }).Desc("Accepts a battle request")
	router.On("gacha", func(ctx *exrouter.Context) { Gacha(ctx) }).Desc("Gacha")
	router.On("servants", func(ctx *exrouter.Context) { checkServants(ctx) }).Desc("Check what servants you have")
	router.Default = router.On("help", func(ctx *exrouter.Context) {
		var a string
		var text string
		a = ctx.Args.Get(1)
		if a == "" {
			text += "```yaml\nScathachBot Commands```"

			for _, v := range router.Routes {
				if v.Name != "bal" && v.Name != "nclaim" && v.Category != "Admin" {
					text += "**" + v.Name + "**" + "\t - \t `" + v.Description + "`\n"
				}
			}
		} else {
			text = "\t\t\t\t\t\t\tAdmin Category\n"
			for _, v := range router.Routes {
				if v.Category == "Admin" {
					text += v.Name + "- \t" + v.Description + "\n"
				}
			}
		}
		text += "```diff\n- Developed By Tsubasa -\n+ " + config.BotPrefix + "suggest [Suggestion] to improve the bot!```\n"
		text += "**Official FGO Fandom Discord\n** <" + "https://discordapp.com/invite/bfRudgg" + ">\n"
		text += "**Official Fandom Site** <" + "https://fategrandorder.fandom.com/wiki/Fate/Grand_Order_Wikia" + ">"
		ctx.ReplyEmbed(16761035, "", text)
	}).Desc("Prints this help menu. " + config.BotPrefix + "help admin for Admin Commands")
}

func Gacha(ctx *exrouter.Context) {
	var servantGot string
	var hasAccountCreated, hasServantCreated bool
	for i := 0; i < len(playerInfo); i++ {
		if playerInfo[i].UserID == ctx.Msg.Author.ID {
			hasAccountCreated = true
			break
		}
	}
	if !hasAccountCreated {
		CreateNewUser(ctx.Msg.Author.ID, ctx.Msg.Author.Username)
	}
	for i := 0; i < len(playerInfo); i++ {
		if playerInfo[i].UserID == ctx.Msg.Author.ID {
			if playerInfo[i].SaintQuartz >= 3 {
				playerInfo[i].SaintQuartz -= 3
				SavePlayerInfo()
				break
			} else {
				ctx.ReplyEmbed(16738657, "Gacha", "You do not have enough <:"+e.APIName()+"> to roll!\n\nYou need **3** <:"+e.APIName()+"> to roll once!")
				return
			}
		}
	}
	for i := 0; i < len(servant); i++ {
		if servant[i].UserID == ctx.Msg.Author.ID {
			hasServantCreated = true
			break
		}
	}
	if !hasServantCreated {
		CreateNewServant(ctx.Msg.Author.ID, ctx.Msg.Author.Username)
	}
	var randNum, randNum2, rarity int
	var thumbnail *discordgo.MessageEmbedThumbnail
	rand.Seed(time.Now().UnixNano())
	randNum = 0
	randNum = rand.Intn(100) + 1
	fmt.Println(randNum)
	if randNum == 50 || randNum == 25 || randNum == 30 || randNum == 45 || randNum == 55 || randNum == 60 || randNum == 65 || randNum == 70 || randNum == 75 || randNum == 80 || randNum == 85 || randNum == 90 || randNum == 95 {
		randNum2 = 0
		randNum2 = rand.Intn(6) + 1
		rarity = 1
		if randNum2 == 1 {
			// Kama
			servantGot = "Kama"
			thumbnail = &discordgo.MessageEmbedThumbnail{
				URL:    "https://vignette.wikia.nocookie.net/fategrandorder/images/1/14/Kama1.png",
				Width: 350,
				Height: 495,
			}
		} else if randNum2 == 2 {
			// Jalter
			servantGot = "Jeanne d'Arc (Alter)"
			thumbnail = &discordgo.MessageEmbedThumbnail{
				URL:    "https://vignette.wikia.nocookie.net/fategrandorder/images/9/98/JeanneAlter1.png",
				Width: 350,
				Height: 495,
			}
		} else if randNum2 == 3 {
			// Arthur
			servantGot = "Arthur Pendragon (Prototype)"
			thumbnail = &discordgo.MessageEmbedThumbnail{
				URL:    "https://vignette.wikia.nocookie.net/fategrandorder/images/2/2b/Arthur1.png",
				Width: 350,
				Height: 495,
			}
		} else if randNum2 == 4 {
			// Jinako
			servantGot = "Great Statue God"
			thumbnail = &discordgo.MessageEmbedThumbnail{
				URL:    "https://vignette.wikia.nocookie.net/fategrandorder/images/b/b7/Jinako1.png",
				Width: 350,
				Height: 495,
			}
		} else if randNum2 == 5 {
			// Scathach
			servantGot = "Scathach"
			thumbnail = &discordgo.MessageEmbedThumbnail{
				URL:    "https://vignette.wikia.nocookie.net/fategrandorder/images/e/e0/Scathach1.png",
				Width: 350,
				Height: 495,
			}
		} else if randNum2 == 6 {
			// Scathach
			servantGot = "Artoria Pendragon"
			thumbnail = &discordgo.MessageEmbedThumbnail{
				URL:    "https://vignette.wikia.nocookie.net/fategrandorder/images/4/43/Artoria1.png",
				Width: 350,
				Height: 495,
			}
		}
	} else {
		randNum2 = 0
		randNum2 = rand.Intn(6) + 1
		fmt.Println(randNum2)
		rarity = 2
		if randNum2 == 1 {
			// Nero
			servantGot = "Nero Claudius"
			thumbnail = &discordgo.MessageEmbedThumbnail{
				URL:    "https://vignette.wikia.nocookie.net/fategrandorder/images/7/71/Nero1.png",
				Width: 350,
				Height: 495,
			}
		} else if randNum2 == 2 {
			// Salter
			servantGot = "Artoria Pendragon (Alter)"
			thumbnail = &discordgo.MessageEmbedThumbnail{
				URL:    "https://vignette.wikia.nocookie.net/fategrandorder/images/a/aa/Alter1.png",
				Width: 350,
				Height: 495,
			}
		} else if randNum2 == 3 {
			// Astolfo
			servantGot = "Astolfo"
			thumbnail = &discordgo.MessageEmbedThumbnail{
				URL:    "https://vignette.wikia.nocookie.net/fategrandorder/images/3/3c/Astolfo1.png",
				Width: 350,
				Height: 495,
			}
		} else if randNum2 == 4 {
			// Heracles
			servantGot = "Heracles"
			thumbnail = &discordgo.MessageEmbedThumbnail{
				URL:    "https://vignette.wikia.nocookie.net/fategrandorder/images/f/f2/Herc1.png",
				Width: 350,
				Height: 495,
			}
		} else if randNum2 == 5 {
			// Nursery Rhyme
			servantGot = "Nursery Rhyme"
			thumbnail = &discordgo.MessageEmbedThumbnail{
				URL:    "https://vignette.wikia.nocookie.net/fategrandorder/images/0/0e/Rhyme1.png",
				Width: 350,
				Height: 495,
			}
		} else if randNum2 == 6 {
			// Medusa Lancer
			servantGot = "Medusa (Lancer)"
			thumbnail = &discordgo.MessageEmbedThumbnail{
				URL: "https://vignette.wikia.nocookie.net/fategrandorder/images/c/c4/Medusalancer1.png",
				Width: 350,
				Height: 495,
			}
		}
	}
	haveServant := dupeServant(ctx.Msg.Author.ID, servantGot, rarity)
		fmt.Println(haveServant)
		Author := &discordgo.MessageEmbedAuthor {
			Name: ctx.Msg.Author.Username +"'s Gacha",
		}
			if haveServant == 1 {
				ctx.ReplyEmbedWithThumbnail(16030402, "★★★★★", Author, thumbnail, "**"+ctx.Msg.Author.Username+"** rolled **" +servantGot+"**!\n\nYou already have **"+servantGot+"**\nReturning **14** <:"+e2.APIName()+"> to you because of duplicate ★★★★★ Servant")
		    } else if haveServant == 2 {
				ctx.ReplyEmbedWithThumbnail(16030402, "★★★★", Author,thumbnail, "**"+ctx.Msg.Author.Username+"** rolled **" +servantGot+"**!\n\nYou already have **"+servantGot+"**\nReturning **7** <:"+e2.APIName()+"> to you because of duplicate ★★★★ Servant")
			} else if haveServant == 3 {
				if rarity == 1 {
					ctx.ReplyEmbedWithThumbnail(16030402, "★★★★★", Author,thumbnail, "**"+ctx.Msg.Author.Username+"** rolled **"+servantGot+"**!")
				} else if rarity == 2 {
					ctx.ReplyEmbedWithThumbnail(16030402, "★★★★", Author,thumbnail, "**"+ctx.Msg.Author.Username+"** rolled **"+servantGot+"**!")
				}
			}
}

func dupeServant(authorid string, servantGot string, servantrarity int) int {
	for i := 0; i < len(servant); i++ {
		if servant[i].UserID == authorid {
			for j := 0; j < len(servant[i].Servants); j++ {
				if servant[i].Servants[j] == servantGot {
					for j := 0; j < len(playerInfo); j++ {
						if playerInfo[j].UserID == authorid {
							if servantrarity == 1 {
								playerInfo[j].SaintQuartzFragment += 14
								SavePlayerInfo()
								return 1
							} else if servantrarity == 2 {
								playerInfo[j].SaintQuartzFragment += 7
								SavePlayerInfo()
								return 2
							}
						}
					}
				}
			}
			servant[i].Servants = append(servant[i].Servants, servantGot)
			SaveServant()
			return 3
		}
	}
	return 0
}

func checkServants(ctx *exrouter.Context) {
	var text string
	if len(ctx.Msg.Mentions) > 0 {
		if len(ctx.Msg.Mentions) >= 2 {
			ctx.ReplyEmbed(16738657, "Invalid Argument", "Syntax: "+config.BotPrefix+"servants /"+config.BotPrefix+"servants [MentionUser]")
		} else if len(ctx.Msg.Mentions) == 1{
			guild, _ := ctx.Guild(ctx.Msg.GuildID)
			roles := guild.Roles
			for i := 0; i < len(roles); i++ {
				if roles[i].ID == ctx.Msg.Mentions[0].ID {
					// Reply that it is an invalid argument since its a role ID and not a user ID
					ctx.ReplyEmbed(16738657, "Invalid Argument", "Syntax: "+config.BotPrefix+"servants /"+config.BotPrefix+"servants [MentionUser]")
					return
				}
			}
			for i := 0; i < len(servant); i++ {
				if servant[i].UserID == ctx.Msg.Mentions[0].ID {
					for j := 0; j < len(servant[i].Servants); j++ {
						text += servant[i].Servants[j] + "\n"
					}
				}
			}
			ctx.ReplyEmbed(16030402, ctx.Msg.Mentions[0].Username+"'s Servants", text)
		}
	} else {
		for i := 0; i < len(servant); i++ {
			if servant[i].UserID == ctx.Msg.Author.ID {
				for j := 0; j < len(servant[i].Servants); j++ {
					text += servant[i].Servants[j] + "\n"
				}
			}
		}
		ctx.ReplyEmbed(16030402, ctx.Msg.Author.Username+"'s Servants", text)
	}
}

func CreateNewServant(newUserID string, newUsername string) {
	newUser := servant[0]
	newUser.Name = newUsername
	newUser.UserID = newUserID
	servant = append(servant, newUser)
}

func ReadServant() error {
	fmt.Println("Reading from servants.json...")
	servantFile, err := ioutil.ReadFile("servants.json")

	if err != nil {
		fmt.Println(err.Error())
	}

	err = json.Unmarshal(servantFile, &servant)

	if err != nil {
		if len(servantFile) == 0 {
			new := servantStruct{
				Name:     "Default",
				UserID:   "DefaultID",
				Servants: []string{},
			}
			servant = append(servant, new)
			file, _ := json.MarshalIndent(battle, "", " ")
			_ = ioutil.WriteFile("servants.json", file, 0644)
		}
	}

	fmt.Println("Finished Reading!")

	return nil
}

func SaveServant() {
	file, _ := json.MarshalIndent(servant, "", " ")
	_ = ioutil.WriteFile("servants.json", file, 0644)
}

func ReadBattle() error {
	fmt.Println("Reading from battle.json...")
	battleFile, err := ioutil.ReadFile("battle.json")

	if err != nil {
		fmt.Println(err.Error())
	}

	err = json.Unmarshal(battleFile, &battle)

	if err != nil {
		if len(battleFile) == 0 {
			new := battleStruct{
				UserID:             "DefaultID",
				Username:           "Default",
				RequesterName:      "Default",
				RequesterID:        "DefaultID",
				RequestedForBattle: false,
				RequestedSomeone:   false,
				InABattle:          false,
				RequestedTime:      time.Now(),
			}
			battle = append(battle, new)
			file, _ := json.MarshalIndent(battle, "", " ")
			_ = ioutil.WriteFile("battle.json", file, 0644)
		}
	}

	fmt.Println("Finished Reading!")

	return nil
}

func SaveBattle() {
	file, _ := json.MarshalIndent(battle, "", " ")
	_ = ioutil.WriteFile("battle.json", file, 0644)
}

func CreateNewBattle(newUserID string, newUsername string) bool {
	for i := 0; i < len(battle); i++ {
		if battle[i].UserID == newUserID {
			return false
		}
	}
	newUser := battle[0]
	newUser.Username = newUsername
	newUser.UserID = newUserID
	newUser.RequesterName = ""
	newUser.RequesterID = ""
	newUser.RequestedName = ""
	newUser.RequestedID = ""
	newUser.RequestedForBattle = false
	newUser.RequestedSomeone = false
	newUser.InABattle = false
	newUser.RequestedTime = time.Now()
	battle = append(battle, newUser)
	return true
}

func checkPvp(s *discordgo.Session, channelID string) { // For time limit of requested time // 1 minute
	for i := 0; i < len(battle); i++ {
		if battle[i].RequestedSomeone {
			var username string
			if time.Since(battle[i].RequestedTime).Minutes() >= 1 {
				for j := 0; j < len(battle); j++ {
					if battle[j].RequesterID == battle[i].UserID {
						username = battle[j].Username
						battle[j].RequestedForBattle = false
						battle[j].RequesterName = ""
						battle[j].RequesterID = ""
						break
					}
				}
				battle[i].RequestedSomeone = false
				battle[i].RequestedName = ""
				battle[i].RequestedID = ""
				SaveBattle()
				embed := discordgo.MessageEmbed{
					Title:       battle[i].Username + "'s Request",
					Description: "Your battle request with " + username + " has expired!",
					Color:       16738657,
				}
				_, _ = s.ChannelMessageSendEmbed(channelID, &embed)
				break
			}
		}
	}
}

func infoBattle(ctx *exrouter.Context) {
	ctx.ReplyEmbed(16030402, "__**Battle Info**__", "**Buster Card** > **Arts Card**\n**Arts Card** > **Quick Card**\n**Quick Card** > **Buster Card**\n**Same Cards = Tie**\nLoser loses **1** <:"+e.APIName()+"> to the Winner\n\n**__Commands__**\n"+config.BotPrefix+"battle [MentionUser] to start a battle with the mentioned user\n *example* "+config.BotPrefix+"battle <@498523735603544096>\n"+config.BotPrefix+"accept to accept any battle requests\nAny battle requests expires in **1** minute if opponent doesn't accept")
	return
}

func reqBattle(ctx *exrouter.Context) { // for requesting a battle
	var hasAccountCreated, hasAccountCreated2 bool // for checking is players has created a playerinfo
	// Check if battle.json has Player's Info and Player Requested Info in it
	if len(ctx.Msg.Mentions) > 0 {
		guild, _ := ctx.Guild(ctx.Msg.GuildID)
		roles := guild.Roles
		for i := 0; i < len(roles); i++ {
			if roles[i].ID == ctx.Msg.Mentions[0].ID {
				// Reply that it is an invalid argument since its a role ID and not a user ID
				ctx.ReplyEmbed(16738657, "Invalid Argument", "Syntax: "+config.BotPrefix+"battle [MentionUser]\n"+config.BotPrefix+"battleinfo for more information")
				return
			}
		}
		// Check if both players has a playerInfo created
		for i := 0; i < len(playerInfo); i++ {
			if playerInfo[i].UserID == ctx.Msg.Author.ID {
				hasAccountCreated = true
			}
			if playerInfo[i].UserID == ctx.Msg.Mentions[0].ID {
				hasAccountCreated2 = true
			}
		}
		if !hasAccountCreated {
			CreateNewUser(ctx.Msg.Author.ID, ctx.Msg.Author.Username)
		}
		if !hasAccountCreated2 {
			CreateNewUser(ctx.Msg.Mentions[0].ID, ctx.Msg.Mentions[0].Username)
		}
		// Check if player requested is not themself
		if ctx.Msg.Mentions[0].ID == ctx.Msg.Author.ID {
			ctx.ReplyEmbed(16738657, ctx.Msg.Author.Username+"'s Battle", "Your greatest enemy is yourself, please choose someone else!")
			return
		}
		// Check if player requested and Player themself has more than or equal to 1 sq
		for i := 0; i < len(playerInfo); i++ {
			if playerInfo[i].UserID == ctx.Msg.Author.ID {
				if playerInfo[i].SaintQuartz >= 1 {
					for j := 0; j < len(playerInfo); j++ {
						if playerInfo[j].UserID == ctx.Msg.Mentions[0].ID {
							if playerInfo[j].SaintQuartz >= 1 {
								break
							} else {
								ctx.ReplyEmbed(16738657, ctx.Msg.Author.Username+"'s Battle", playerInfo[j].Name+" does not have 1 <:"+e.APIName()+"> to battle!")
								return
							}
						}
					}
				} else {
					ctx.ReplyEmbed(16738657, ctx.Msg.Author.Username+"'s Battle", "You do not have 1 <:"+e.APIName()+"> to battle!")
					return
				}
			}
		}
		// If its not a role and if its not themself then it will create
		CreateNewBattle(ctx.Msg.Author.ID, ctx.Msg.Author.Username)
		CreateNewBattle(ctx.Msg.Mentions[0].ID, ctx.Msg.Mentions[0].Username)
	} else {
		// If there is no mentions
		ctx.ReplyEmbed(16738657, "Invalid Argument", "Syntax: "+config.BotPrefix+"battle [MentionUser]\n"+config.BotPrefix+"battleinfo for more information")
		return
	}
	// Check if you are currently requested by someone
	for i := 0; i < len(battle); i++ {
		if battle[i].UserID == ctx.Msg.Author.ID {
			if battle[i].RequestedForBattle {
				if battle[i].RequesterID == ctx.Msg.Mentions[0].ID {
					ctx.ReplyEmbed(16738657, ctx.Msg.Author.Username+"'s Battle", "You have already been requested by "+battle[i].RequesterName+"\n Please do s!accept to accept their request")
					return
				} else {
					ctx.ReplyEmbed(16738657, ctx.Msg.Author.Username+"'s Battle", "You have already been requested by "+battle[i].RequesterName+"\n Please do s!accept to accept their request")
					return
				}
			} else {
				break
			}
		}
	}
	// Check if Player already requested someone for battle or check if Player already requested player being requested for battle
	for i := 0; i < len(battle); i++ {
		if battle[i].UserID == ctx.Msg.Author.ID {
			if battle[i].RequestedSomeone {
				for j := 0; j < len(battle); j++ {
					if battle[j].UserID == ctx.Msg.Mentions[0].ID {
						if battle[j].RequesterID == ctx.Msg.Author.ID {
							ctx.ReplyEmbed(16738657, ctx.Msg.Author.Username+"'s Battle", "You already requested a battle to "+battle[j].Username+"!")
							return
						}
					}
				}
				ctx.ReplyEmbed(16738657, ctx.Msg.Author.Username+"'s Battle", "You already requested a battle with "+battle[i].RequestedName)
				return
			} else {
				break
			}
		}
	}
	// Check if player being requested has already been requested by someone else
	for i := 0; i < len(battle); i++ {
		if battle[i].UserID == ctx.Msg.Mentions[0].ID {
			if battle[i].RequestedForBattle {
				ctx.ReplyEmbed(16738657, ctx.Msg.Author.Username+"'s Battle", ctx.Msg.Mentions[0].Username+" is already requested for a battle by "+battle[i].RequesterName)
				return
			} else {
				break
			}
		}
	}
	// Check if player being requested is in a battle
	for i := 0; i < len(battle); i++ {
		if battle[i].UserID == ctx.Msg.Mentions[0].ID {
			if battle[i].InABattle {
				ctx.ReplyEmbed(16738657, ctx.Msg.Author.Username+"'s Battle", ctx.Msg.Mentions[0].Username+" is already in a battle!")
				return
			} else {
				break
			}
		}
	}
	// Set up for acceptance of battle
	//UserID             string    // Personal ID
	//Username           string    // Requester's ID
	//RequesterName      string    // Requester's Name
	//RequesterID        string    // Requester's ID
	//RequestedName      string    // Requested's Name
	//RequestedID        string    // Requested's ID
	//RequestedForBattle bool      // Check if the player is already requested for a battle
	//InABattle          bool      // Check if the player is in a battle
	//RequestedSomeone   bool      // Checks if the player requested a battle with someone
	//RequestedTime      time.Time // Time when they requested a User
	for i := 0; i < len(battle); i++ {
		if battle[i].UserID == ctx.Msg.Author.ID {
			battle[i].RequestedName = ctx.Msg.Mentions[0].Username
			battle[i].RequestedID = ctx.Msg.Mentions[0].ID
			battle[i].RequestedSomeone = true
			battle[i].RequestedTime = time.Now()
		}
		if battle[i].UserID == ctx.Msg.Mentions[0].ID {
			battle[i].RequesterName = ctx.Msg.Author.Username
			battle[i].RequesterID = ctx.Msg.Author.ID
			battle[i].RequestedForBattle = true
		}
	}
	SaveBattle()
	ctx.ReplyEmbed(11075425, ctx.Msg.Author.Username+"'s Battle", "You have successfully sent a battle request to "+ctx.Msg.Mentions[0].Username+"\n"+ctx.Msg.Mentions[0].Username+" has to type "+config.BotPrefix+"accept to accept the battle request")
	return
}

func acceptBattle(ctx *exrouter.Context) {
	var p1, p1id string
	hasABattleInfo := CreateNewBattle(ctx.Msg.Author.ID, ctx.Msg.Author.Username)
	// Player does not even have an account created
	// hasABattleInfo == true means does not have account created
	if hasABattleInfo {
		ctx.ReplyEmbed(16738657, ctx.Msg.Author.Username+"'s Battle", "You currently do not have any battle requests!")
		return
	} else {
		// check if player has a battle request
		for i := 0; i < len(battle); i++ {
			if battle[i].UserID == ctx.Msg.Author.ID {
				if battle[i].RequestedForBattle {
					p1 = battle[i].RequesterName
					p1id = battle[i].RequesterID
					battle[i].InABattle = true
					for j := 0; j < len(battle); j++ {
						if battle[j].UserID == p1id {
							battle[j].InABattle = true
							battle[j].RequestedSomeone = false
							battle[j].RequestedName = ""
							battle[j].RequestedID = ""
						}
					}
					battle[i].RequesterName = ""
					battle[i].RequesterID = ""
					battle[i].RequestedForBattle = false
					break
				} else {
					ctx.ReplyEmbed(16738657, ctx.Msg.Author.Username+"'s Battle", "You currently do not have any battle requests!")
					return
				}
			}
		}
		//  Battle
		var Player1, Player2 int
		rand.Seed(time.Now().UnixNano())
		Player1 = 0
		Player2 = 0
		Player1 = rand.Intn(3) + 1 // Player 1 Requester
		Player2 = rand.Intn(3) + 1 // Player 2 NonRequester
		// 1 = Buster
		// 2 = Arts
		// 3 = Quick
		// Buster > Arts, Arts > Quick, Quick > Buster
		p2 := ctx.Msg.Author.Username
		p2id := ctx.Msg.Author.ID
		if Player1 == 1 && Player2 == 1 { // tied
			ctx.ReplyEmbed(16030402, "Battle", p1+" drawed a **Buster** Card\n"+p2+" drawed a **Buster** Card\nBoth players tied!")
			for i := 0; i < len(battle); i++ {
				if battle[i].UserID == p1id {
					battle[i].InABattle = false
				}
				if battle[i].UserID == p2id {
					battle[i].InABattle = false
				}
			}
			SaveBattle()
			return
		} else if Player1 == 2 && Player2 == 2 { // tied
			ctx.ReplyEmbed(16030402, "Battle", p1+" drawed an **Arts** Card\n"+p2+" drawed an **Arts** Card\nBoth players tied!")
			for i := 0; i < len(battle); i++ {
				if battle[i].UserID == p1id {
					battle[i].InABattle = false
				}
				if battle[i].UserID == p2id {
					battle[i].InABattle = false
				}
			}
			SaveBattle()
			return
		} else if Player1 == 3 && Player2 == 3 { // tied
			ctx.ReplyEmbed(16030402, "Battle", p1+" drawed a **Quick** Card\n"+p2+" drawed a **Quick** Card\nBoth players tied!")
			for i := 0; i < len(battle); i++ {
				if battle[i].UserID == p1id {
					battle[i].InABattle = false
				}
				if battle[i].UserID == p2id {
					battle[i].InABattle = false
				}
			}
			SaveBattle()
			return
		} else if Player1 == 1 && Player2 == 2 { // Player1 Buster > Arts
			ctx.ReplyEmbed(16030402, "Battle", p1+" drawed a **Buster** Card\n"+p2+" drawed an **Arts** Card\n"+p1+" won the battle!")
			for i := 0; i < len(playerInfo); i++ {
				if playerInfo[i].UserID == p1id {
					playerInfo[i].SaintQuartz += 1
				}
				if playerInfo[i].UserID == p2id {
					playerInfo[i].SaintQuartz -= 1
				}
			}
			for i := 0; i < len(battle); i++ {
				if battle[i].UserID == p1id {
					battle[i].InABattle = false
				}
				if battle[i].UserID == p2id {
					battle[i].InABattle = false
				}
			}
			SavePlayerInfo()
			SaveBattle()
			return
		} else if Player1 == 1 && Player2 == 3 { // Player2 Quick > Buster
			ctx.ReplyEmbed(16030402, "Battle", p1+" drawed a **Buster** Card\n"+p2+" drawed a **Quick** Card\n"+p2+" won the battle!")
			for i := 0; i < len(playerInfo); i++ {
				if playerInfo[i].UserID == p1id {
					playerInfo[i].SaintQuartz -= 1
				}
				if playerInfo[i].UserID == p2id {
					playerInfo[i].SaintQuartz += 1
				}
			}
			for i := 0; i < len(battle); i++ {
				if battle[i].UserID == p1id {
					battle[i].InABattle = false
				}
				if battle[i].UserID == p2id {
					battle[i].InABattle = false
				}
			}
			SavePlayerInfo()
			SaveBattle()
			return
		} else if Player2 == 1 && Player1 == 2 { // Player2 Buster > Arts
			ctx.ReplyEmbed(16030402, "Battle", p1+" drawed a **Buster** Card\n"+p2+" drawed an **Arts** Card\n"+p2+" won the battle!")
			for i := 0; i < len(playerInfo); i++ {
				if playerInfo[i].UserID == p1id {
					playerInfo[i].SaintQuartz -= 1
				}
				if playerInfo[i].UserID == p2id {
					playerInfo[i].SaintQuartz += 1
				}
			}
			for i := 0; i < len(battle); i++ {
				if battle[i].UserID == p1id {
					battle[i].InABattle = false
				}
				if battle[i].UserID == p2id {
					battle[i].InABattle = false
				}
			}
			SavePlayerInfo()
			SaveBattle()
			return
		} else if Player2 == 1 && Player1 == 3 { // Player1 Quick > Buster
			ctx.ReplyEmbed(16030402, "Battle", p1+" drawed a **Quick** Card\n"+p2+" drawed a **Buster** Card\n"+p1+" won the battle!")
			for i := 0; i < len(playerInfo); i++ {
				if playerInfo[i].UserID == p1id {
					playerInfo[i].SaintQuartz += 1
				}
				if playerInfo[i].UserID == p2id {
					playerInfo[i].SaintQuartz -= 1
				}
			}
			for i := 0; i < len(battle); i++ {
				if battle[i].UserID == p1id {
					battle[i].InABattle = false
				}
				if battle[i].UserID == p2id {
					battle[i].InABattle = false
				}
			}
			SavePlayerInfo()
			SaveBattle()
			return
		} else if Player1 == 2 && Player2 == 3 { // Player1 Arts > Quick
			ctx.ReplyEmbed(16030402, "Battle", p1+" drawed an **Arts** Card\n"+p2+" drawed a **Quick** Card\n"+p1+" won the battle!")
			for i := 0; i < len(playerInfo); i++ {
				if playerInfo[i].UserID == p1id {
					playerInfo[i].SaintQuartz += 1
				}
				if playerInfo[i].UserID == p2id {
					playerInfo[i].SaintQuartz -= 1
				}
			}
			for i := 0; i < len(battle); i++ {
				if battle[i].UserID == p1id {
					battle[i].InABattle = false
				}
				if battle[i].UserID == p2id {
					battle[i].InABattle = false
				}
			}
			SavePlayerInfo()
			SaveBattle()
			return
		} else if Player2 == 2 && Player1 == 3 { // Player2 Arts > Quick
			ctx.ReplyEmbed(16030402, "Battle", p1+" drawed a **Qquick** Card\n"+p2+" drawed an **Arts** Card\n"+p2+" won the battle!")
			for i := 0; i < len(playerInfo); i++ {
				if playerInfo[i].UserID == p1id {
					playerInfo[i].SaintQuartz -= 1
				}
				if playerInfo[i].UserID == p2id {
					playerInfo[i].SaintQuartz += 1
				}
			}
			for i := 0; i < len(battle); i++ {
				if battle[i].UserID == p1id {
					battle[i].InABattle = false
				}
				if battle[i].UserID == p2id {
					battle[i].InABattle = false
				}
			}
			SavePlayerInfo()
			SaveBattle()
			return
		}
	}
}

func shuffle(text string) string {
	newText := strings.Split(text, "")
	var x int
	x = len(newText)
	for i := 0; i < x; i++ {
		n := rand.Intn(x)
		x := newText[i]
		newText[i] = newText[n]
		newText[n] = x
	}
	returnText := strings.Join(newText, " ")
	returnText2 := strings.Trim(returnText, " ")
	return returnText2
}

func exchangeSQF(ctx *exrouter.Context) {
	var x int
	for i := 0; i < len(playerInfo); i++ {
		if playerInfo[i].UserID == ctx.Msg.Author.ID {
			if playerInfo[i].SaintQuartzFragment >= 7 {
				for playerInfo[i].SaintQuartzFragment >= 7 {
					playerInfo[i].SaintQuartzFragment = playerInfo[i].SaintQuartzFragment - 7
					playerInfo[i].SaintQuartz = playerInfo[i].SaintQuartz + 1
					x++
				}
				SavePlayerInfo()
				xStr := strconv.Itoa(x)
				ctx.ReplyEmbed(16738657, "<:"+e2.APIName()+"> → "+"<:"+e.APIName()+">", "Successfully exchanged "+xStr+" set(s) of <:"+e2.APIName()+"> to "+xStr+" <:"+e.APIName()+">")
				return
			} else {
				ctx.ReplyEmbed(8782097, "", "You do not have enough <:"+e2.APIName()+"> to exchange")
				return
			}
		}
	}
	fmt.Println("Creating New User...")
	CreateNewUser(ctx.Msg.Author.ID, ctx.Msg.Author.Username)
	SavePlayerInfo()
	ctx.ReplyEmbed(8782097, "", "You do not have enough <:"+e2.APIName()+"> to exchange")
}

func scrambleString(inStr string) string {
	//make new array of runes of the same size
	out := make([]rune, len(inStr))
	//ensure randomness
	rand.Seed(time.Now().UnixNano())

	for _, inCharacter := range inStr {
		for {
			randomOutIndex := rand.Intn(len(inStr))
			if out[randomOutIndex] == 0 {
				out[randomOutIndex] = inCharacter
				break
			}
		}
	}
	return string(out)
}

func dice(ctx *exrouter.Context) {
	rand.Seed(time.Now().UTC().UnixNano())
	random := rand.Intn(6) + 1
	randomstr := strconv.Itoa(random)
	ctx.ReplyEmbed(16777215, "Dice", "You rolled a "+randomstr)
}

func miniGame(s *discordgo.Session, channelID string) {
	var secondTime float64
	var secondTimeInt int
	if !miniGameStart {
		return
	}
	if !once {
		once = true
		// timeInterval = time.Now()
	}
	//time.Since(timeInterval).Seconds() >= 5 && 
	if once {
		if !sendOnce {
			sendOnce = true
			rand.Seed(time.Now().UnixNano())

			RandNum := rand.Intn(177)
			Running = 1

			if RandNum == 0 {
				RandNum = 1
			}

			if line, err := rsl("words.txt", RandNum); err == nil {
				Original = strings.TrimSpace(line)
				randomizedStr := scrambleString(Original)
				embed := discordgo.MessageEmbed{
					Title:       "Unscramble",
					Description: "```css\n" + randomizedStr + "```\n _Reward_ \t\t\t" + "**1** <:" + e2.APIName() + ">",
					Color:       16757575,
				}
				timeSolve = time.Now()
				_, _ = s.ChannelMessageSendEmbed(channelID, &embed)
			} else {
				fmt.Println(err.Error)
			}
		} else {
			secondTimeInt = 0
			secondTime = 0
			secondTime = 60.0 - time.Since(timeSolve).Seconds()
			secondTimeInt = int(secondTime) % 60
			secondStr := strconv.Itoa(secondTimeInt)
			fmt.Println(secondTime)
			if secondTimeInt == 59 {
				embed := discordgo.MessageEmbed{
					Description: "You have **1** minute to solve the minigame!",
					Color:       65280,
				}
				_, _ = s.ChannelMessageSendEmbed(channelID, &embed)
			}
			if secondTimeInt == 10 {
				embed := discordgo.MessageEmbed{
					Description: "You have **" + secondStr + "** seconds left to solve the minigame!",
					Color:       16729344,
				}
				_, _ = s.ChannelMessageSendEmbed(channelID, &embed)
			}
			if secondTimeInt <= 0 {
				Running = 0
				once = false
				sendOnce = false
				miniGameStart = false
				solved = false
				embed := discordgo.MessageEmbed{
					Description: "The answer was, **" + Original + "**",
					Color:       16711680,
				}
				_, _ = s.ChannelMessageSendEmbed(channelID, &embed)
			}
		}
	}
}

func shop(ctx *exrouter.Context) {

	file, err := ioutil.ReadFile("shop.txt")

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	ctx.ReplyEmbed(16761035, "Shop", string(file))
}

func rsl(fn string, n int) (string, error) {
	if n < 1 {
		return "", fmt.Errorf("invalid request: line %d", n)
	}
	f, err := os.Open(fn)
	if err != nil {
		return "", err
	}
	defer f.Close()
	bf := bufio.NewReader(f)
	var line string
	for lnum := 0; lnum < n; lnum++ {
		line, err = bf.ReadString('\n')
		if err == io.EOF {
			switch lnum {
			case 0:
				return "", errors.New("no lines in file")
			case 1:
				return "", errors.New("only 1 line")
			default:
				return "", fmt.Errorf("only %d lines", lnum)
			}
		}
		if err != nil {
			return "", err
		}
	}
	if line == "" {
		return "", fmt.Errorf("line %d empty", n)
	}
	return line, nil
}

func prefix(ctx *exrouter.Context) {
	var a string
	a = ctx.Args.Get(1)
	if a == "" {
		ctx.Reply("Prefix cannot be blank!")
		return
	} else {
		fmt.Println("Saving new prefix...")
		configFile := configStruct{
			Token:     config.Token,
			BotPrefix: a,
		}
		ctx.Reply("Changed your prefix to " + a)

		file, _ := json.MarshalIndent(&configFile, "", " ")
		_ = ioutil.WriteFile("config.json", file, 0644)
	}
}

func suggest(ctx *exrouter.Context) {
	var a string
	var output string
	var guild *discordgo.Guild
	a = ctx.Args.After(1)
	if a == "" {
		for i := 0; i < len(suggestion); i++ {
			if suggestion[i].UserID == ctx.Msg.Author.ID {
				for j := 0; j < len(suggestion[i].Suggestion); j++ {
					num := strconv.Itoa(j + 1)
					output += "**" + num + ".**"
					output += suggestion[i].Suggestion[j] + "\n"
				}
			}
		}
		ctx.ReplyEmbed(7855479, "__**Your Current Suggestions**__", output)
		return
	} else {
		guild, _ = ctx.Guild(ctx.Msg.GuildID)

		fmt.Println("You received a new suggestion from " + ctx.Msg.Author.Username + " in " + guild.Name)
		for i := 0; i < len(suggestion); i++ {
			if suggestion[i].UserID == ctx.Msg.Author.ID {
				for j := 0; j < len(suggestion[i].Server); j++ {
					if suggestion[i].Server[j] != guild.Name {
						suggestion[i].Server = append(suggestion[i].Server, guild.Name)
					}
				}
				suggestion[i].Suggestion = append(suggestion[i].Suggestion, a)
				ctx.ReplyEmbed(16738657, "Suggestion", "You suggested:"+"```diff\n +"+a+"```")
				SaveSuggestion()
				return
			}
		}

		new := suggestion[0]
		new.Username = ctx.Msg.Author.Username
		new.Server = append(new.Server, guild.Name)
		new.UserID = ctx.Msg.Author.ID
		new.Suggestion = append(new.Suggestion, a)
		suggestion = append(suggestion, new)
		ctx.ReplyEmbed(16738657, "Suggestion", "You suggested:"+"```"+a+"```")
		SaveSuggestion()
		return
	}
}

func askSuggest(ctx *exrouter.Context) {
	a, _ := strconv.Atoi(ctx.Args.Get(1))
	if a >= 1 {
		a--
		suggestionNum = a
		for i := 0; i < len(suggestion); i++ {
			if suggestion[i].UserID == ctx.Msg.Author.ID {
				if !checker {
					suggestionChannel = ctx.Msg.ChannelID
					suggestionID = ctx.Msg.Author.ID
					suggestionUser = i
					ctx.ReplyEmbed(8782097, "", "Are you sure you want to remove:```diff\n- "+suggestion[i].Suggestion[a]+"```")
					checker = true
				}
			}
		}
	} else {
		ctx.ReplyEmbed(8782097, "Invalid Argument", "Syntax: "+config.BotPrefix+"rsuggest [SuggestionNumber]")
		return
	}
}

func ReadSuggestion() error {
	fmt.Println("Reading from suggestion.json...")
	suggestionFile, err := ioutil.ReadFile("suggestion.json")

	if err != nil {
		fmt.Println(err.Error())
	}

	err = json.Unmarshal(suggestionFile, &suggestion)

	if err != nil {
		if len(suggestionFile) == 0 {
			new := suggestionStruct{
				Username:   "Default",
				Server:     []string{},
				UserID:     "Default",
				Suggestion: []string{},
			}
			suggestion = append(suggestion, new)
			file, _ := json.MarshalIndent(suggestion, "", " ")
			_ = ioutil.WriteFile("suggestion.json", file, 0644)
		}
	}

	fmt.Println("Finished Reading!")

	return nil
}

func SaveSuggestion() {
	file, _ := json.MarshalIndent(suggestion, "", " ")
	_ = ioutil.WriteFile("suggestion.json", file, 0644)
}

func balance(ctx *exrouter.Context) {
	var userTarget string
	var userName string
	if len(ctx.Msg.Mentions) > 0 {
		userTarget = ctx.Msg.Mentions[0].ID
		userName = ctx.Msg.Mentions[0].Username
	} else {
		userTarget = ctx.Msg.Author.ID
		userName = ctx.Msg.Author.Username
	}
	for i := 0; i < len(playerInfo); i++ {
		if playerInfo[i].UserID == userTarget {
			var bal = strconv.Itoa(playerInfo[i].SaintQuartz)
			var bal2 = strconv.Itoa(playerInfo[i].SaintQuartzFragment)
			if userTarget == ctx.Msg.Author.ID {
				if playerInfo[i].Booster > 0 {
					var boost = strconv.Itoa(playerInfo[i].Booster)
					ctx.ReplyEmbed(10494192, ctx.Msg.Author.Username+"'s Balance", "**"+bal+"** <:"+e.APIName()+">\n**"+bal2+"** <:"+e2.APIName()+">\n"+"+ **"+boost+"** _Booster(s)_!")
				} else {
					ctx.ReplyEmbed(10494192, ctx.Msg.Author.Username+"'s Balance", "**"+bal+"** <:"+e.APIName()+">\n**"+bal2+"** <:"+e2.APIName()+">")
				}
			} else {
				if playerInfo[i].Booster > 0 {
					var boost = strconv.Itoa(playerInfo[i].Booster)
					ctx.ReplyEmbed(10494192, ctx.Msg.Mentions[0].Username+"'s Balance", "**"+bal+"** <:"+e.APIName()+">\n**"+bal2+"** <:"+e2.APIName()+">\n"+"**"+boost+"** _Booster(s)_!")
				} else {
					ctx.ReplyEmbed(10494192, ctx.Msg.Mentions[0].Username+"'s Balance", "**"+bal+"** <:"+e.APIName()+">\n**"+bal2+"** <:"+e2.APIName()+">")
				}
			}
			return
		}
	}
	fmt.Println("Creating New User...")
	CreateNewUser(userTarget, userName)
	var bal = strconv.Itoa(playerInfo[len(playerInfo)-1].SaintQuartz)
	var bal2 = strconv.Itoa(playerInfo[len(playerInfo)-1].SaintQuartzFragment)
	if userTarget == ctx.Msg.Author.ID {
		ctx.ReplyEmbed(10494192, ctx.Msg.Author.Username+"'s Balance", "**"+bal+"** <:"+e.APIName()+">\n**"+bal2+"** <:"+e2.APIName()+">")
	} else {
		ctx.ReplyEmbed(10494192, ctx.Msg.Mentions[0].Username+"'s Balance", "**"+bal+"** <:"+e.APIName()+">\n**"+bal2+"** <:"+e2.APIName()+">")
	}
	SavePlayerInfo()
	return
}

func resetBalance(ctx *exrouter.Context) {
	var userTarget string
	var userName string
	var clearAll bool
	a := ctx.Args.Get(1)
	if len(ctx.Msg.Mentions) > 0 {
		userTarget = ctx.Msg.Mentions[0].ID
		userName = ctx.Msg.Mentions[0].Username
	} else {
		if a == "all" {
			clearAll = true
		} else {
			userTarget = ctx.Msg.Author.ID
			userName = ctx.Msg.Author.Username
		}
	}
	if !clearAll {
		for i := 0; i < len(playerInfo); i++ {
			if playerInfo[i].UserID == userTarget {
				playerInfo[i].SaintQuartz = 0
				if userTarget == ctx.Msg.Author.ID {
					ctx.Reply("Reset " + ctx.Msg.Author.Mention() + "'s <:" + e.APIName() + "> &" + " <:" + e2.APIName() + "> to 0")
				} else {
					ctx.Reply("Reset " + ctx.Msg.Mentions[0].Mention() + "'s <:" + e.APIName() + "> &" + " <:" + e2.APIName() + "> to 0")
				}
				SavePlayerInfo()
				return
			}
		}
		fmt.Println("Creating New User...")
		CreateNewUser(userTarget, userName)
		SavePlayerInfo()
	} else {
		for i := 0; i < len(playerInfo); i++ {
			playerInfo[i].SaintQuartz = 0
		}
		SavePlayerInfo()
		ctx.Reply("Successfully reset everyone's <:" + e.APIName() + "> &" + " <:" + e2.APIName() + "> to 0")
		clearAll = false
	}

}

func buyRole(s *discordgo.Session, ctx *exrouter.Context) {
	var a, b string
	var color int
	a = ctx.Args.Get(1)
	b = ctx.Args.Get(2)
	color, _ = strconv.Atoi(b)
	for i := 0; i < len(playerInfo); i++ {
		if playerInfo[i].UserID == ctx.Msg.Author.ID {
			if playerInfo[i].SaintQuartz >= 500 {
				playerInfo[i].SaintQuartz = playerInfo[i].SaintQuartz - 500
				SavePlayerInfo()
				ctx.Reply("Successfully bought your role: " + a + ". -500 <:" + e.APIName() + ">")
				guildrole, _ := s.GuildRoleCreate(ctx.Msg.GuildID)
				guild, err := s.GuildRoleEdit(ctx.Msg.GuildID, guildrole.ID, a, color, false, 0, false)
				if err != nil {
					ctx.Reply(err.Error())
				}
				s.GuildMemberRoleAdd(ctx.Msg.GuildID, ctx.Msg.Author.ID, guild.ID)
			} else {
				ctx.Reply("You do not have enough <:" + e.APIName() + "> to purchase this")
			}
		}
	}

}

func calc(duration time.Duration) (second, minute, hour, day int) {
	var seconds = int(duration.Seconds())
	second = seconds % 60
	minute = seconds / 60 % 60
	hour = seconds / 60 / 60 % 24
	day = seconds / 60 / 60 / 24
	return
}

func boosterExpiry(s *discordgo.Session, m *discordgo.MessageCreate) {
	for i := 0; i < len(playerInfo); i++ {
		if playerInfo[i].BoosterUsed == true {
			if time.Since(playerInfo[i].UsedBooster).Hours() >= 10 {
				playerInfo[i].BoosterUsed = false
			}
		}
	}
}

func useBooster(ctx *exrouter.Context) {
	var hasNitroBoost bool
	var roleID string
	userTarget := ctx.Msg.Author.ID
	hasNitroBoost = false
	member, _ := ctx.Member(ctx.Msg.GuildID, ctx.Msg.Author.ID)
	guild, _ := ctx.Guild(ctx.Msg.GuildID)
	roles := guild.Roles
	for i := 0; i < len(roles); i++ {
		if roles[i].Name == "Nitro Booster" {
			roleID = roles[i].ID
		}
	}
	for _, role := range member.Roles {
		if role == roleID {
			hasNitroBoost = true
		}
	}
	if hasNitroBoost {
		for i := 0; i < len(playerInfo); i++ {
			if playerInfo[i].UserID == userTarget {
				if playerInfo[i].Booster > 0 {
					if !playerInfo[i].BoosterUsed {
						playerInfo[i].Booster -= 1
						playerInfo[i].UsedBooster = time.Now()
						playerInfo[i].BoosterUsed = true
						ctx.ReplyEmbed(16773120, "Nitro Booster", "You used your _Booster_!")
						SavePlayerInfo()
						return
					} else {
						d := (10 * time.Hour)
						second, minute, hour, _ := calc(d - time.Since(playerInfo[i].UsedBooster))
						seconds := strconv.Itoa(second)
						minutes := strconv.Itoa(minute)
						hours := strconv.Itoa(hour)
						ctx.ReplyEmbed(8782097, "Nitro Booster", "Your _Booster_ is already activated!\n"+"Your _Booster_ has **"+hours+"** hour **"+minutes+"** minutes **"+seconds+"** seconds left before expiry")
						return
					}
				} else {
					if playerInfo[i].BoosterUsed {
						d := (10 * time.Hour)
						second, minute, hour, _ := calc(d - time.Since(playerInfo[i].UsedBooster))
						seconds := strconv.Itoa(second)
						minutes := strconv.Itoa(minute)
						hours := strconv.Itoa(hour)
						ctx.ReplyEmbed(8782097, "Nitro Booster", "Your _Booster_ is already activated!\n"+"Your _Booster_ has **"+hours+"** hour **"+minutes+"** minutes **"+seconds+"** seconds left before expiry")
						return
					} else {
						ctx.ReplyEmbed(8782097, "Nitro Booster", "You do not have any _Boosters_!")
						return
					}
				}
			}
		}
	} else {
		ctx.ReplyEmbed(10658512, "", "You do not have _Nitro Booster_ role!")
		return
	}

}

func nitroclaim(ctx *exrouter.Context) {
	var hasNitroBoost bool
	var roleID string
	userTarget := ctx.Msg.Author.ID
	userName := ctx.Msg.Author.Username
	hasNitroBoost = false
	member, _ := ctx.Member(ctx.Msg.GuildID, ctx.Msg.Author.ID)
	guild, _ := ctx.Guild(ctx.Msg.GuildID)
	roles := guild.Roles
	for i := 0; i < len(roles); i++ {
		if roles[i].Name == "Nitro Booster" {
			roleID = roles[i].ID
		}
	}
	for _, role := range member.Roles {
		if role == roleID {
			hasNitroBoost = true
		}
	}
	for i := 0; i < len(playerInfo); i++ {
		if hasNitroBoost {
			if playerInfo[i].UserID == ctx.Msg.Author.ID {
				if time.Since(playerInfo[i].LastCollectedNitro).Hours() >= 72 {
					playerInfo[i].Booster += 1
					ctx.ReplyEmbed(16023551, "Nitro Booster", "You claimed your _Booster_ for the minigame")
					playerInfo[i].LastCollectedNitro = time.Now()
					SavePlayerInfo()
					return
				} else {
					d := (72 * time.Hour)
					second, minute, hour, day := calc(d - time.Since(playerInfo[i].LastCollectedNitro))
					seconds := strconv.Itoa(second)
					minutes := strconv.Itoa(minute)
					hours := strconv.Itoa(hour)
					days := strconv.Itoa(day)
					ctx.ReplyEmbed(8782097, "Nitro Booster", "You can collect your _Booster_ in **"+days+"** days **"+hours+"** hours **"+minutes+"** minutes and **"+seconds+"** seconds")
					return
				}
			}
		} else {
			ctx.ReplyEmbed(10658512, "", "You do not have _Nitro Booster_ role to claim this")
			return
		}
	}
	fmt.Println("Creating New User...")
	CreateNewUser(userTarget, userName)
	for i := 0; i < len(playerInfo); i++ {
		if hasNitroBoost {
			if playerInfo[i].UserID == ctx.Msg.Author.ID {
				if time.Since(playerInfo[i].LastCollectedNitro).Hours() >= 72 {
					playerInfo[i].Booster += 1
					ctx.ReplyEmbed(16023551, "Nitro Booster", "You claimed your _Booster_ for the minigame")
					playerInfo[i].LastCollectedNitro = time.Now()
					SavePlayerInfo()
					return
				}
			}
		} else {
			ctx.ReplyEmbed(10658512, "", "You do not have _Nitro Booster_ role to claim this")
			return
		}
	}
}

func ResetDaily() {
	var currentTime time.Time
	loc := resetTime.Location()
	currentTime = time.Now().In(loc)
	if currentTime.Hour() == resetTime.Hour() && currentTime.Minute() == resetTime.Minute() && currentTime.Second() == resetTime.Second() {
		for i := 0; i < len(playerInfo); i++ {
			playerInfo[i].DailyCollect = true
		}
		resetTime = time.Date(timeNow.Year(), timeNow.Month(), timeNow.Day()+1, 24, 00, 00, 00, time.FixedZone("UTC+9", 9*60*60))
	}
}

func daily(ctx *exrouter.Context) {
	var currentTime time.Time
	var hasNitroBoost bool
	var roleID string
	userTarget := ctx.Msg.Author.ID
	userName := ctx.Msg.Author.Username
	hasNitroBoost = false
	loc := resetTime.Location()
	currentTime = time.Now().In(loc)
	member, _ := ctx.Member(ctx.Msg.GuildID, ctx.Msg.Author.ID)
	guild, _ := ctx.Guild(ctx.Msg.GuildID)
	roles := guild.Roles
	for i := 0; i < len(roles); i++ {
		if roles[i].Name == "Nitro Booster" {
			roleID = roles[i].ID
		}
	}
	for _, role := range member.Roles {
		if role == roleID {
			hasNitroBoost = true
		}
	}
	for i := 0; i < len(playerInfo); i++ {
		if playerInfo[i].UserID == ctx.Msg.Author.ID {
			if playerInfo[i].DailyCollect {
				if hasNitroBoost {
					playerInfo[i].DailyCollect = false
					playerInfo[i].SaintQuartz += 2
					playerInfo[i].BonusDay += 1
					ctx.ReplyEmbed(16023551, "Nitro Booster", "You have claimed 2 <:"+e.APIName()+"> because you Nitro Boosted the server!")
					SavePlayerInfo()
					return
				} else {
					playerInfo[i].DailyCollect = false

					playerInfo[i].BonusDay += 1
					if playerInfo[i].BonusDay == 7 {
						playerInfo[i].BonusDay = 0
						playerInfo[i].SaintQuartz += 3
						ctx.ReplyEmbed(9212588, "Daily", "You have claimed 3 <:"+e.APIName()+"> because you hit 7 days of daily")
					} else {
						playerInfo[i].SaintQuartz += 1
						ctx.ReplyEmbed(9212588, "Daily", "You have claimed 1 <:"+e.APIName()+">")
					}

					SavePlayerInfo()
					return
				}
			} else {
				second, minute, hour, _ := calc(time.Since(currentTime) - time.Since(resetTime))
				hours := strconv.Itoa(hour)
				minutes := strconv.Itoa(minute)
				seconds := strconv.Itoa(second)
				ctx.ReplyEmbed(8782097, "Daily", "<:"+e.APIName()+"> Resets in **"+hours+"** hours **"+minutes+"** minutes and **"+seconds+"** seconds\n"+"\n__*Daily resets at 12am Japan Standard Time*__")
				return
			}
		}
	}
	fmt.Println("Creating New User...")
	CreateNewUser(userTarget, userName)
	for i := 0; i < len(playerInfo); i++ {
		if playerInfo[i].UserID == ctx.Msg.Author.ID {
			if playerInfo[i].DailyCollect {
				if hasNitroBoost {
					playerInfo[i].DailyCollect = false
					playerInfo[i].SaintQuartz += 2
					playerInfo[i].BonusDay += 1
					ctx.ReplyEmbed(16023551, "Nitro Booster", "You have claimed 2 <:"+e.APIName()+"> because you Nitro Boosted the server!")
					SavePlayerInfo()
					return
				} else {
					playerInfo[i].DailyCollect = false
					playerInfo[i].SaintQuartz += 1
					playerInfo[i].BonusDay += 1
					ctx.ReplyEmbed(9212588, "Daily", "You have claimed 1 <:"+e.APIName()+">")
					SavePlayerInfo()
					return
				}
			}
		}
	}
}

func addBalance(ctx *exrouter.Context) {
	var userTarget, userName, a string

	if len(ctx.Msg.Mentions) > 0 {
		userTarget = ctx.Msg.Mentions[0].ID
		userName = ctx.Msg.Mentions[0].Username
		a = ctx.Args.Get(2)
	} else {
		userTarget = ctx.Msg.Author.ID
		userName = ctx.Msg.Author.Username
		a = ctx.Args.Get(1)
	}
	var amount, err = strconv.Atoi(a)
	if err != nil {
		ctx.Reply("Please enter a valid argument! Example: " + config.BotPrefix + "addbalance [User] [Amount]")
		return
	}
	for i := 0; i < len(playerInfo); i++ {
		if playerInfo[i].UserID == userTarget {
			playerInfo[i].SaintQuartz += amount
			if userTarget == ctx.Msg.Author.ID {
				ctx.Reply("Gave " + a + " <:" + e.APIName() + "> to " + ctx.Msg.Author.Mention())
			} else {
				ctx.Reply("Gave " + a + " <:" + e.APIName() + "> to " + ctx.Msg.Mentions[0].Mention())
			}
			SavePlayerInfo()
			return
		}
	}
	fmt.Println("Creating New User...")
	CreateNewUser(userTarget, userName)
	for i := 0; i < len(playerInfo); i++ {
		if playerInfo[i].UserID == userTarget {
			playerInfo[i].SaintQuartz += amount
			if userTarget == ctx.Msg.Author.ID {
				ctx.Reply("Gave " + a + " <:" + e.APIName() + "> to " + ctx.Msg.Author.Mention())
			} else {
				ctx.Reply("Gave " + a + " <:" + e.APIName() + "> to " + ctx.Msg.Mentions[0].Mention())
			}
			SavePlayerInfo()
			return
		}
	}
	return
}

func Auth(fn exrouter.HandlerFunc) exrouter.HandlerFunc {
	return func(ctx *exrouter.Context) {
		var roleID string
		var hasOwner bool
		var hasRole bool
		member, err := ctx.Member(ctx.Msg.GuildID, ctx.Msg.Author.ID)
		if err != nil {
			ctx.ReplyEmbed(8782097, "Could not fetch member:", err)
			return
		}
		guild, err := ctx.Guild(ctx.Msg.GuildID)
		if err != nil {
			ctx.ReplyEmbed(8782097, "Could not fetch Guild:", err)
			return
		}
		roles := guild.Roles
		for i := 0; i < len(roles); i++ {
			if roles[i].Name == "ScathachAdmin" {
				roleID = roles[i].ID
				hasRole = true
			}
		}
		if !hasRole {
			ctx.ReplyEmbed(8782097, "**Could not find role named. Please create ScathachAdmin**")
			return
		}
		for _, role := range member.Roles {
			if role == roleID {
				ctx.Set("Owner", member)
				fn(ctx)
				hasOwner = true
				return
			} else {
				hasOwner = false
			}
		}
		if !hasOwner {
			ctx.ReplyEmbed(8782097, "**You need ScathachAdmin role to use this command**")
			return
		}
	}
}

func CreateNewUser(newUserID string, newUsername string) {
	newUser := playerInfo[0]
	newUser.Name = newUsername
	newUser.UserID = newUserID
	newUser.SaintQuartz = 0
	newUser.SaintQuartzFragment = 0
	newUser.BonusDay = 0
	newUser.Booster = 0
	newUser.BoosterUsed = false
	newUser.DailyCollect = true
	newUser.LastCollectedNitro = time.Now().Local().AddDate(0, 0, -3)
	newUser.UsedBooster = time.Now()
	playerInfo = append(playerInfo, newUser)
}

func ReadPlayerInfo() error {

	fmt.Println("Reading from playerinfo.json...")

	playerInfofile, err := ioutil.ReadFile("playerinfo.json")

	if err != nil {
		fmt.Println(err.Error())
	}

	err = json.Unmarshal(playerInfofile, &playerInfo)

	if err != nil {
		fmt.Println("No Users in playerinfo.json. Creating default user...")
		if len(playerInfo) == 0 {
			newUser := playerInfoStruct{
				Name:                "Default",
				UserID:              "Default",
				SaintQuartz:         0,
				SaintQuartzFragment: 0,
				BonusDay:            0,
				Booster:             0,
				DailyCollect:        true,
				LastCollectedNitro:  time.Now(),
				UsedBooster:         time.Now(),
			}
			playerInfo = append(playerInfo, newUser)
			SavePlayerInfo()
		}
		fmt.Println("Finished creating default user!")
	}

	fmt.Println("Finished Reading!")

	return nil

}

func SavePlayerInfo() error {
	fmt.Println("Saving to playerinfo file...")

	file, _ := json.MarshalIndent(playerInfo, "", " ")
	_ = ioutil.WriteFile("playerinfo.json", file, 0644)

	return nil
}
