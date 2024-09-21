package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const inputDir = "ArticleMerge"            // 存放要合并的文本文件的目录
const outputDir = "ArticleComplete"        // 保存完整文章的目录


func readAPIKeyFromFile(filename string) (string, error) {
    file, err := os.Open(filename)
    if err != nil {
        return "", err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    if scanner.Scan() {
        return scanner.Text(), nil
    }

    if err := scanner.Err(); err != nil {
        return "", err
    }

    return "", fmt.Errorf("file is empty")
}

// Function to call OpenAI API to merge and refine the content into a complete article
func mergeAndRefineContent(contents string, openaiAPIKey string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	reqBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "system", "content": "你是一個幫助用戶撰寫文章的助手。請將以下內容整理成一篇流暢且完整的文章。"},
			{"role": "user", "content": fmt.Sprintf("請將以下內容整理成一篇完整的文章:\n\n%s", contents)},
		},
		"max_tokens": 1500, // 增加 max_tokens 值，以生成更長的文章
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openaiAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Debug: Print out the full API response
	fmt.Println("OpenAI API response:", string(body))

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("unexpected API response format or no choices found")
	}

	message, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected format for the message field in API response")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected format for the content field in API response")
	}

	return content, nil
}

// Function to read all .txt files in the input directory and merge their content
func readAndMergeFiles(inputDir string) (string, error) {
	var mergedContent strings.Builder

	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
			fmt.Printf("Reading file: %s\n", path)

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			mergedContent.WriteString(string(content) + "\n\n")
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return mergedContent.String(), nil
}

// Function to process files, merge their content, and save the completed article
func processAndSaveArticle(openaiAPIKey string) error {
	fmt.Println("Starting to process and merge files...")

	// Read and merge all .txt files from the input directory
	mergedContent, err := readAndMergeFiles(inputDir)
	if err != nil {
		return err
	}

	fmt.Println("Merged content:", mergedContent)

	// Send the merged content to OpenAI API for refinement
	completeArticle, err := mergeAndRefineContent(mergedContent, openaiAPIKey)
	if err != nil {
		return err
	}

	// Save the completed article to the output directory
	err = os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return err
	}

	outputFileName := filepath.Join(outputDir, "complete_article.txt")
	err = ioutil.WriteFile(outputFileName, []byte(completeArticle), 0644)
	if err != nil {
		return err
	}

	fmt.Printf("Saved complete article to: %s\n", outputFileName)
	return nil
}

func main() {
	// Read the API key from the file
	openaiAPIKey, err := readAPIKeyFromFile("openaikey")
	if err != nil {
		fmt.Println("Error reading API key:", err)
		return
	}

	// Use the openaiAPIKey variable as needed
	fmt.Println("OpenAI API Key:", openaiAPIKey)

	err = processAndSaveArticle(openaiAPIKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
