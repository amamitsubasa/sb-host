package main

import (
	"fmt"

	"./bot"
	"./config"
)

func main() {
	err := config.ReadConfig()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	bot.Start()

	err = bot.ReadPlayerInfo()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = bot.ReadSuggestion()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = bot.ReadBattle()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = bot.ReadServant()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for {
		bot.ResetDaily()
	}

	<-make(chan struct{})
	return
}
