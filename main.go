package main

import (
    "io/ioutil"
    "log"
    "net/http"

    "github.com/gin-gonic/gin"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

type WorkProject struct {
	ID                uint   `gorm:"primaryKey"`
    ProjectName       string `json:"project_name"`
    ProjectPictures   []byte `json:"project_pictures"`
    ProjectDescription string `json:"project_description"`
}

func (WorkProject) TableName() string {
    return "workproject"
}

var DB *gorm.DB


func main() {
	dsn := "host=localhost user=postgres password=bumi2698 dbname=portofolio-apps port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	var err error

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
        log.Fatal("Failed to connect to database: ", err)
    }

	r := gin.Default()

	r.GET("/workprojects", GetWorkProjects)
	r.POST("/workprojects", CreateWorkProject)
     r.POST("/upload", func(c *gin.Context) {
        handleFileUpload(c, DB)
    })

    r.Run(":8085")
}

func GetWorkProjects(c *gin.Context) {
    var workprojects []WorkProject
    result := DB.Find(&workprojects)
	response := gin.H{
		"data": workprojects,
		"isError": result.Error != nil,
	}

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, response)
		return
	}

    c.JSON(http.StatusOK, response)
}

func handleFileUpload(c *gin.Context, db *gorm.DB) {
    file, _, err := c.Request.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file"})
        return
    }
    defer file.Close()

    fileData, err := ioutil.ReadAll(file)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
        return
    }

    result := db.Exec("INSERT INTO workproject_images (image_data) VALUES ($1)", fileData)
    if result.Error != nil {
        log.Println("Database insert error:", result.Error)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert image into database"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Image uploaded successfully"})
}

func CreateWorkProject(c *gin.Context) {
    err := c.Request.ParseMultipartForm(10 << 20)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form data"})
        return
    }
    projectName := c.Request.FormValue("project_name")
    projectDescription := c.Request.FormValue("project_description")
    file, _, err := c.Request.FormFile("project_pictures")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file"})
        return
    }
    defer file.Close()

    fileData, err := ioutil.ReadAll(file)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
        return
    }

    project := WorkProject{
        ProjectName:        projectName,
        ProjectPictures:    fileData,
        ProjectDescription: projectDescription,
    }

    result := DB.Create(&project)
    if result.Error != nil {
        log.Println("Database insert error:", result.Error)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert work project into database"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Work project created successfully", "data": project})
}