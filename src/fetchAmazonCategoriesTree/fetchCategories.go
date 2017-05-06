package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Category struct {
	title, link string
}

type Task struct {
	link                string
	categoriesHierarchy []string
}

var retries int = 30
var startLink string = "https://www.amazon.co.uk/gp/bestsellers/books"
var workersCount int = 1 << 5
var tasksCount int64 = 0
var outputFile *os.File

var taskChannel chan *Task = make(chan *Task, workersCount*10000)
var resultWriteLock sync.Mutex = sync.Mutex{}

var httpClient *http.Client = &http.Client{}

func getCategoriesFromPage(link string) []*Category {

	request, _ := http.NewRequest("GET", link, nil)

	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.81 Safari/537.36")

	response, err := httpClient.Do(request)

	if err != nil {
		fmt.Println("Error:", err.Error())
		return nil
	}

	document, parseErr := goquery.NewDocumentFromResponse(response)

	if parseErr != nil {
		fmt.Println("Error:", parseErr.Error())
		return nil
	}

	return getAllCategories(document)
}

func writeResults(categoriesList []string) {
	resultWriteLock.Lock()
	outputFile.WriteString("[" + strings.Join(categoriesList, ",") + "],")
	resultWriteLock.Unlock()
}

func buildCategoriesTree(link string, categoriesHierarchy []string) {
	categoriesList := getCategoriesFromPage(link)

	if categoriesList != nil && len(categoriesList) > 0 {

		for _, category := range categoriesList {
			newList := make([]string, len(categoriesHierarchy))
			copy(newList, categoriesHierarchy)
			newList = append(newList, category.title)

			taskChannel <- &Task{
				link:                category.link,
				categoriesHierarchy: newList,
			}
		}
	} else {
		writeResults(categoriesHierarchy)
	}

}

func handleTasks(workerIndex int) {
	for {
		select {
		case newTask := <-taskChannel:
			//fmt.Println("Worker", workerIndex, ": running New Tasks")
			atomic.AddInt64(&tasksCount, 1)
			buildCategoriesTree(newTask.link, newTask.categoriesHierarchy)
			atomic.AddInt64(&tasksCount, -1)
		}
	}
}

func main() {

	var err error
	outputFile, err = os.OpenFile("resultsList.json", os.O_CREATE|os.O_WRONLY, 0664)

	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}

	defer outputFile.Close()
	outputFile.WriteString("[")

	for workersCount > 0 {
		func(workerIndex int) {
			go handleTasks(workerIndex)
		}(workersCount)

		workersCount -= 1
	}

	taskChannel <- &Task{
		link:                startLink,
		categoriesHierarchy: make([]string, 0),
	}

	ticker := time.NewTicker(time.Second * 1)

	var totalPendingTasksCount int64

	for {
		select {
		case <-ticker.C:
		}

		totalPendingTasksCount = atomic.LoadInt64(&tasksCount)

		if totalPendingTasksCount <= 0 {
			retries -= 1
			if retries == 0 {
				break
			}
		} else {
			retries = 30
			//fmt.Println("Total Tasks:", totalPendingTasksCount)
		}
	}

	_, err = outputFile.WriteString("]")

	if err != nil {
		fmt.Println("Error:", err.Error())
	}
}
