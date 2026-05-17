package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"
)

type RawItem struct {
	Type    string   `json:"type"`
	User    string   `json:"user"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

type TrainingItem struct {
	Instruction string `json:"instruction"`
	Input       string `json:"input"`
	Output      string `json:"output"`
}

const (
	InputFile  = "input.txt"
	OutputFile = "dataset.jsonl"
)

func main() {

	file, err := os.Open(InputFile)
	if err != nil {
		fmt.Printf("❌ Нет файла %s. Создайте его и вставьте JSON.\n", InputFile)
		return
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)
	content := string(bytes)

	content = strings.TrimSpace(content)
	content = strings.ReplaceAll(content, "```json", "")
	content = strings.ReplaceAll(content, "```", "")

	var rawItems []RawItem
	if err := json.Unmarshal([]byte(content), &rawItems); err != nil {
		fmt.Printf("❌ Ошибка JSON: %v\n", err)
		return
	}

	outFile, err := os.OpenFile(OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	encoder := json.NewEncoder(outFile)
	count := 0

	for _, item := range rawItems {
		var finalOutput string
		thought := generateThought(item.Type)

		switch item.Type {
		case "chat":

			finalOutput = item.Content
		case "search":

			finalOutput = fmt.Sprintf("МЫСЛЬ: %s\nCMD:SEARCH | %s", thought, item.Content)
		case "google":

			finalOutput = fmt.Sprintf("МЫСЛЬ: %s\nCMD:GOOGLE | %s", thought, item.Content)
		case "save":

			tagsStr := strings.Join(item.Tags, ", ")
			finalOutput = fmt.Sprintf("МЫСЛЬ: %s\nCMD:SAVE | %s | %s", thought, item.Content, tagsStr)
		}

		trainItem := TrainingItem{
			Instruction: item.User,
			Input:       "",
			Output:      finalOutput,
		}

		if err := encoder.Encode(trainItem); err != nil {
			fmt.Printf("Ошибка записи: %v\n", err)
			continue
		}
		count++
	}

	_ = os.WriteFile(InputFile, []byte(""), 0644)
	fmt.Printf("✅ Добавлено %d строк в %s. Input очищен.\n", count, OutputFile)
}

func generateThought(msgType string) string {
	thoughts := map[string][]string{
		"search": {
			"Нужно проверить в памяти.",
			"Пользователь спрашивает о прошлом. Ищу.",
			"Кажется, я это запоминал. Проверяю базу.",
			"Вопрос касается личных данных. Делаю запрос.",
		},
		"google": {
			"Это вопрос о внешнем мире. Гуглю.",
			"Нужна актуальная информация из сети.",
			"Проверю новости и факты в интернете.",
			"В моей памяти этого нет, но есть в Google.",
		},
		"save": {
			"Пользователь сообщил важный факт. Сохраняю.",
			"Нужно запомнить эту информацию.",
			"Обновляю профиль пользователя.",
			"Это пригодится в будущем. Записываю.",
		},
	}

	list, exists := thoughts[msgType]
	if !exists {
		return ""
	}
	rand.Seed(time.Now().UnixNano())
	return list[rand.Intn(len(list))]
}
