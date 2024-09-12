package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const openaiAPIKey = "" // 替换为你的 OpenAI API Key
const searchResultDir = "ArticleFetching"  // BingSearch 生成的链接文件的目录
const outputDir = "ArticleDownload"        // 摘要文件的保存目录

// Function to read the API key from a file
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

// Function to call OpenAI API to summarize text in Chinese
func summarizeText(text string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	reqBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "system", "content": "你是一個幫助用戶總結文章的助手。請用中文總結以下文章，並確保總結字數接近300字。"},
			{"role": "user", "content": fmt.Sprintf("請總結以下文章的主要內容:\n\n%s", text)},
		},
		"max_tokens": 600,
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

// Function to download article content from URL with improved error handling
func downloadArticle(url string) (string, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")
	req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch the URL: %s, status code: %d", url, resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	// Extract the main content of the article
	var articleText string
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		articleText += s.Text() + "\n"
	})

	return articleText, nil
}

// Function to process each search result file and summarize articles
func processSearchResults() error {
	fmt.Println("Starting to process search result files...")
	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return err
	}

	err = filepath.Walk(searchResultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
			fmt.Printf("Processing search result file: %s\n", info.Name())

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			links := strings.Split(string(content), "\n")
			for i, link := range links {
				link = strings.TrimSpace(link)
				if link != "" {
					fmt.Printf("Downloading article from: %s\n", link)
					articleText, err := downloadArticle(link)
					if err != nil {
						fmt.Printf("Failed to download article from: %s, error: %v\n", link, err)
						continue
					}

					// Limit the article text length for summarization
					if len(articleText) > 4000 {
						articleText = articleText[:4000]
					}

					fmt.Println("Summarizing article...")
					summary, err := summarizeText(articleText)
					if err != nil {
						fmt.Printf("Failed to summarize article from: %s, error: %v\n", link, err)
						continue
					}

					outputFileName := filepath.Join(outputDir, fmt.Sprintf("%s_%d.txt", strings.TrimSuffix(info.Name(), ".txt"), i+1))
					err = ioutil.WriteFile(outputFileName, []byte(summary), 0644)
					if err != nil {
						return err
					}

					fmt.Printf("Saved summary to: %s\n", outputFileName)
				}
			}
		}

		return nil
	})

	return err
}

func main() {
	// Read the API key from the file
	openaiAPIKey, err := readAPIKeyFromFile("key")
	if err != nil {
		fmt.Println("Error reading API key:", err)
		return
	}

	// Use the openaiAPIKey variable as needed
	fmt.Println("OpenAI API Key:", openaiAPIKey)

	err := processSearchResults()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
