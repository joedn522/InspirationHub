package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const openaiAPIKey = "sk-WNsjjCTIQPGo-qTbVDp1-vEJdor81fWjlutf6LoUvWT3BlbkFJFXyc0N3aTSLXrCybbPIz447IOE9byIQKeku_t_QBcA" // 在此處填入你的 API Key
const outputDir = "textDir"               // 輸出 metadata 的資料夾

// Function to call OpenAI API to get keywords from text
func getKeywordsFromLLM(text string) ([]string, error) {
	url := "https://api.openai.com/v1/completions"
	reqBody := map[string]interface{}{
		"model": "text-davinci-003",
		"prompt": fmt.Sprintf(
			"Extract keywords from the following text:\n%s",
			text,
		),
		"max_tokens": 50,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openaiAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	keywordsRaw := result["choices"].([]interface{})[0].(map[string]interface{})["text"].(string)
	keywords := strings.Split(strings.TrimSpace(keywordsRaw), ",")
	for i := range keywords {
		keywords[i] = strings.TrimSpace(keywords[i])
	}

	return keywords, nil
}

// Function to process all txt files in a directory
func processFiles(inputDir string) error {
	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return err
	}

	err = filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			keywords, err := getKeywordsFromLLM(string(content))
			if err != nil {
				return err
			}

			outputFileName := filepath.Join(outputDir, info.Name()+".metadata")
			metadataContent := strings.Join(keywords, ", ")

			err = ioutil.WriteFile(outputFileName, []byte(metadataContent), 0644)
			if err != nil {
				return err
			}

			fmt.Printf("Processed file: %s\n", path)
		}

		return nil
	})

	return err
}

func main() {
	inputDir := "textDir" // 替換為你的資料夾路徑
	err := processFiles(inputDir)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}