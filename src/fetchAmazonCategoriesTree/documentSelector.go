package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"strconv"
)

func getSelectedCategory(document *goquery.Document) (selectedCategoryNode *goquery.Selection) {
	return document.Find("ul ul li span").First()
}

func getAllCategories(document *goquery.Document) (categoriesList []*Category) {

	selectedCategoryNode := getSelectedCategory(document)

	childCategoriesContainer := selectedCategoryNode.Parent().Parent().Find("ul").First()

	if len(childCategoriesContainer.Nodes) > 0 {

		categoriesList = make([]*Category, 0)

		childCategoriesContainer.Find("li a").Each(func(_ int, selection *goquery.Selection) {

			categoryLink, exists := selection.Attr("href")

			if exists {
				newCategory := &Category{
					link:  categoryLink,
					title: strconv.Quote(selection.Text()),
				}

				categoriesList = append(categoriesList, newCategory)
			} else {
				fmt.Println("Href for category doesn't exists")
			}
		})
	}
	return
}
