package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type WorkProject struct {
	ID                uint   `gorm:"primaryKey"`
    ProjectName       string `json:"project_name"`
    ProjectPictures   []byte `json:"project_pictures"`
    ProjectDescription string `json:"project_description"`
}

type ProjectDetailPayload struct {
    ProjectFeaturesName       string `json:"project_features_name"`
    ProjectFeaturesPicture    string `json:"project_features_picture"`
    ProjectFeaturesDescription string `json:"project_features_description"`
}

type ProjectPayload struct {
    ProjectName        string               `json:"project_name"`
    ProjectBasePicture string               `json:"project_base_picture"`
    ProjectDescription string               `json:"project_description"`
    ProjectDetailed    []ProjectDetailPayload `json:"project_detailed"`
}

type ProjectDetail struct {
    ProjectFeaturesName       string `json:"project_features_name"`
    ProjectFeaturesPicture    string `json:"project_features_picture"`
    ProjectFeaturesDescription string `json:"project_features_description"`
}

type ProjectList struct {
    ProjectName         string           `json:"project_name"`
    ProjectBasePicture  []byte           `json:"project_base_picture"`
    ProjectDescription  string           `json:"project_description"`
    ProjectDetailed     []ProjectDetail  `json:"project_detailed"`
}

func (WorkProject) TableName() string {
    return "workproject"
}

var DB *gorm.DB

var dbconnection = "host=localhost user=postgres password=bumi2698 dbname=portofolio-apps port=5432 sslmode=disable TimeZone=Asia/Shanghai"


func main() {
	var err error

	DB, err = gorm.Open(postgres.Open(dbconnection), &gorm.Config{})

	if err != nil {
        log.Fatal("Failed to connect to database: ", err)
    }

	r := gin.Default()

	r.GET("/workprojects", GetWorkProjects)
    r.GET("/workprojects/detail/", GetProjectsDetail)
	r.POST("/workprojects", CreateWorkProject)
    r.POST("/workprojects/detail/post", PostProjectDetail)
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

func PostProjectDetail(c *gin.Context) {
    db, err := sql.Open("postgres", dbconnection)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
        return
    }
    defer db.Close()

    var projectPayload ProjectPayload
    if err := c.BindJSON(&projectPayload); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
        return
    }

    // Decode Base64 image for project_base_picture
    basePictureData, err := base64.StdEncoding.DecodeString(projectPayload.ProjectBasePicture)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Base64 for project_base_picture"})
        return
    }

    // Insert into project_list
    var projectId int
    err = db.QueryRow(`
        INSERT INTO project_list (project_name, project_base_picture, project_description)
        VALUES ($1, $2, $3) RETURNING id
    `, projectPayload.ProjectName, basePictureData, projectPayload.ProjectDescription).Scan(&projectId)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert project"})
        return
    }

    // Insert project_detailed data
    for _, detail := range projectPayload.ProjectDetailed {
        // Decode Base64 picture
        pictureData, err := base64.StdEncoding.DecodeString(detail.ProjectFeaturesPicture)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Base64 for project_features_picture"})
            return
        }

        _, err = db.Exec(`
            INSERT INTO project_detail (parent_id, project_features_name, project_features_picture, project_features_description)
            VALUES ($1, $2, $3, $4)
        `, projectId, detail.ProjectFeaturesName, pictureData, detail.ProjectFeaturesDescription)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert project detail"})
            return
        }
    }

    c.JSON(http.StatusOK, gin.H{"message": "Project created successfully"})
}

func GetProjectsDetail(c *gin.Context) {
    db, err := sql.Open("postgres", dbconnection)
    if err != nil {
        c.JSON(500, gin.H{"error": "Database connection error"})
        return
    }
    defer db.Close()

    rows, err := db.Query(`
        SELECT
            p.project_name,
            p.project_base_picture,
            p.project_description,
            json_agg(
                json_build_object(
                    'project_features_name', d.project_features_name,
                    'project_features_picture', encode(d.project_features_picture, 'base64'),
                    'project_features_description', d.project_features_description
                )
            ) AS project_detailed
        FROM
            project_list p
        LEFT JOIN
            project_detail d ON p.id = d.parent_id
        GROUP BY
            p.id;
    `)
    if err != nil {
        c.JSON(500, gin.H{"error": "Query execution error"})
        return
    }
    defer rows.Close()

    var projects []ProjectList

    var isError bool = false

    for rows.Next() {
        var project ProjectList
        var detailed json.RawMessage
        err := rows.Scan(&project.ProjectName, &project.ProjectBasePicture, &project.ProjectDescription, &detailed)
        if err != nil {
            isError = true
        }
        json.Unmarshal(detailed, &project.ProjectDetailed)
        projects = append(projects, project)
    }

    response := gin.H{
		"data": projects,
		"isError": isError,
	}

	if isError {
		c.JSON(http.StatusInternalServerError, response)
		return
	}

    c.JSON(http.StatusOK, response)
}