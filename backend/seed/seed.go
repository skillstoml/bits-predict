package seed

import (
	"fmt"
	"os"

	configsvc "socialpredict/internal/service/config"
	"socialpredict/logger"
	"socialpredict/models"
	"time"

	"socialpredict/handlers/cms/homepage"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func getEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("environment variable %s not set", key)
	}
	return value, nil
}

func SeedUsers(db *gorm.DB, configService configsvc.Service) error {
	if configService == nil {
		return fmt.Errorf("configuration service unavailable")
	}
	userConfig := configService.Economics().User

	adminPassword, err := getEnv("ADMIN_PASSWORD")
	if err != nil {
		return fmt.Errorf("retrieving ADMIN_PASSWORD: %w", err)
	}
	if adminPassword == "" {
		return fmt.Errorf("ADMIN_PASSWORD is set but empty")
	}

	var count int64
	if err := db.Model(&models.User{}).Where("username = ?", "admin").Count(&count).Error; err != nil {
		return fmt.Errorf("check admin user: %w", err)
	}
	if count == 0 {
		adminUser := models.User{
			PublicUser: models.PublicUser{
				Username:              "admin",
				DisplayName:           "Administrator",
				UserType:              "ADMIN",
				InitialAccountBalance: userConfig.InitialAccountBalance,
				AccountBalance:        userConfig.InitialAccountBalance,
				PersonalEmoji:         "NONE",
				Description:           "Administrator",
			},
			PrivateUser: models.PrivateUser{
				Email:  "admin@example.com",
				APIKey: "NONE",
			},
			MustChangePassword: true,
		}

		if err := adminUser.HashPassword(adminPassword); err != nil {
			return fmt.Errorf("hash admin password: %w", err)
		}

		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "username"}},
			DoNothing: true,
		}).Create(&adminUser).Error; err != nil {
			return fmt.Errorf("create admin user: %w", err)
		}
	}

	return nil
}

func EnsureDBReady(db *gorm.DB, maxAttempts int) error {
	var err error
	for attempts := 1; attempts <= maxAttempts; attempts++ {
		// Attempt to perform a simple operation like pinging the database
		sqlDB, err := db.DB()
		if err != nil {
			logger.LogWarn("Seed", "EnsureDBReady", "unable to get database/sql DB from GORM DB: "+err.Error())
			time.Sleep(time.Second * 5)
			continue
		}

		err = sqlDB.Ping()
		if err != nil {
			logger.LogWarn("Seed", "EnsureDBReady", fmt.Sprintf("failed to connect to the database, attempt %d/%d: %v", attempts, maxAttempts, err))
			time.Sleep(time.Second * 5) // Wait before retrying
			continue
		}

		logger.LogInfo("Seed", "EnsureDBReady", "Database is ready.")
		return nil
	}

	return fmt.Errorf("database is not ready after %d attempts: %v", maxAttempts, err)
}

func SeedHomepage(db *gorm.DB, repoRoot string) error {
	var count int64
	if err := db.Model(&models.HomepageContent{}).
		Where("slug = ?", "home").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	// Use embedded content to avoid filesystem path issues
	var data []byte
	if len(defaultHomeMD) > 0 {
		data = defaultHomeMD
	} else {
		// Fallback only if embedding failed
		data = []byte("# Welcome to BrierFoxForecast\n\nThis is the seeded home page.")
	}

	// Create renderer for sanitization
	renderer := homepage.NewDefaultRenderer()

	// Since the content is pure HTML, treat it as HTML format
	htmlContent := string(data)

	// Sanitize the HTML directly (no markdown conversion needed)
	sanitizedHTML := renderer.SanitizeHTML(htmlContent)

	item := models.HomepageContent{
		Slug:     "home",
		Title:    "Home",
		Format:   "html", // Changed to html since content is pure HTML
		Markdown: "",     // Empty since we're using HTML format
		HTML:     sanitizedHTML,
		Version:  1,
	}

	return db.Create(&item).Error
}

func SeedMarkets(db *gorm.DB) error {
	// 1. Check if the markets are already seeded
	var count int64
	if err := db.Model(&models.MarketGroup{}).Where("question_title LIKE ?", "%BITS Pilani%").Count(&count).Error; err != nil {
		return fmt.Errorf("checking market groups: %w", err)
	}
	if count > 0 {
		logger.LogInfo("Seed", "SeedMarkets", "BITS Pilani markets already seeded.")
		return nil
	}

	// 2. We need an admin user to own the markets
	var adminCount int64
	if err := db.Model(&models.User{}).Where("username = ?", "admin").Count(&adminCount).Error; err != nil {
		return fmt.Errorf("checking admin user: %w", err)
	}
	if adminCount == 0 {
		return fmt.Errorf("admin user not found, seed users first")
	}

	now := time.Now()
	resolutionTime := now.AddDate(0, 1, 0) // 1 month in the future

	// Create President Election Group
	prezGroup := models.MarketGroup{
		QuestionTitle:      "Who will be elected BITS Pilani SU President?",
		Description:        "This market group resolves to the candidate who wins the BITS Pilani Student Union President election.",
		GroupType:          "MULTIPLE_CHOICE_BINARY",
		ProbabilityPolicy:  "INDEPENDENT_BINARY",
		ResolutionPolicy:   "INDEPENDENT_CHILDREN",
		LifecycleStatus:    "published",
		ProposalCost:       0,
		CreatorUsername:    "admin",
		StewardUsername:    "admin",
		ApprovedBy:         "admin",
		ApprovedAt:         &now,
		ResolutionDateTime: resolutionTime,
	}

	if err := db.Create(&prezGroup).Error; err != nil {
		return fmt.Errorf("creating prez market group: %w", err)
	}

	prezCandidates := []string{
		"Anirudh Jyothiraditya Panguluri",
		"Dhaval Rakesh Bothra",
		"Pulkit Bhardwaj",
		"Vansh Malik",
		"None of the Above",
	}
	for i, name := range prezCandidates {
		childMarket := models.Market{
			QuestionTitle:      fmt.Sprintf("Who will be elected BITS Pilani SU President? - %s", name),
			Description:        fmt.Sprintf("Will %s win the BITS Pilani Student Union President election?", name),
			OutcomeType:        "BINARY",
			ResolutionDateTime: resolutionTime,
			IsResolved:         false,
			InitialProbability: 0.5,
			YesLabel:           "YES",
			NoLabel:            "NO",
			LifecycleStatus:    "published",
			ApprovedBy:         "admin",
			ApprovedAt:         &now,
			CreatorUsername:    "admin",
			StewardUsername:    "admin",
		}
		if err := db.Create(&childMarket).Error; err != nil {
			return fmt.Errorf("creating prez child market %s: %w", name, err)
		}

		member := models.MarketGroupMember{
			GroupID:      prezGroup.ID,
			MarketID:     childMarket.ID,
			AnswerLabel:  name,
			DisplayOrder: i,
		}
		if err := db.Create(&member).Error; err != nil {
			return fmt.Errorf("creating prez market group member %s: %w", name, err)
		}
	}

	// Create GenSec Election Group
	gensecGroup := models.MarketGroup{
		QuestionTitle:      "Who will be elected BITS Pilani SU GenSec?",
		Description:        "This market group resolves to the candidate who wins the BITS Pilani Student Union General Secretary election.",
		GroupType:          "MULTIPLE_CHOICE_BINARY",
		ProbabilityPolicy:  "INDEPENDENT_BINARY",
		ResolutionPolicy:   "INDEPENDENT_CHILDREN",
		LifecycleStatus:    "published",
		ProposalCost:       0,
		CreatorUsername:    "admin",
		StewardUsername:    "admin",
		ApprovedBy:         "admin",
		ApprovedAt:         &now,
		ResolutionDateTime: resolutionTime,
	}

	if err := db.Create(&gensecGroup).Error; err != nil {
		return fmt.Errorf("creating gensec market group: %w", err)
	}

	gensecCandidates := []string{
		"Kushal Poosala",
		"Moksh Goel",
		"Pratyush Goenka",
		"Rajay Vardhan Rai",
		"None of the Above",
	}
	for i, name := range gensecCandidates {
		childMarket := models.Market{
			QuestionTitle:      fmt.Sprintf("Who will be elected BITS Pilani SU GenSec? - %s", name),
			Description:        fmt.Sprintf("Will %s win the BITS Pilani Student Union General Secretary election?", name),
			OutcomeType:        "BINARY",
			ResolutionDateTime: resolutionTime,
			IsResolved:         false,
			InitialProbability: 0.5,
			YesLabel:           "YES",
			NoLabel:            "NO",
			LifecycleStatus:    "published",
			ApprovedBy:         "admin",
			ApprovedAt:         &now,
			CreatorUsername:    "admin",
			StewardUsername:    "admin",
		}
		if err := db.Create(&childMarket).Error; err != nil {
			return fmt.Errorf("creating gensec child market %s: %w", name, err)
		}

		member := models.MarketGroupMember{
			GroupID:      gensecGroup.ID,
			MarketID:     childMarket.ID,
			AnswerLabel:  name,
			DisplayOrder: i,
		}
		if err := db.Create(&member).Error; err != nil {
			return fmt.Errorf("creating gensec market group member %s: %w", name, err)
		}
	}

	// Create Day Scholar Representative (D-Rep) Group
	drepGroup := models.MarketGroup{
		QuestionTitle:      "Who will be elected BITS Pilani Day Scholar Representative (D-Rep)?",
		Description:        "This market group resolves to the candidate who wins the BITS Pilani Student Union Day Scholar Representative (D-Rep) election.",
		GroupType:          "MULTIPLE_CHOICE_BINARY",
		ProbabilityPolicy:  "INDEPENDENT_BINARY",
		ResolutionPolicy:   "INDEPENDENT_CHILDREN",
		LifecycleStatus:    "published",
		ProposalCost:       0,
		CreatorUsername:    "admin",
		StewardUsername:    "admin",
		ApprovedBy:         "admin",
		ApprovedAt:         &now,
		ResolutionDateTime: resolutionTime,
	}

	if err := db.Create(&drepGroup).Error; err != nil {
		return fmt.Errorf("creating drep market group: %w", err)
	}

	drepCandidates := []string{
		"Krishna Sharma",
		"Mohit Singh Bhati",
		"Vansh Lahora",
		"None of the Above",
	}
	for i, name := range drepCandidates {
		childMarket := models.Market{
			QuestionTitle:      fmt.Sprintf("Who will be elected BITS Pilani Day Scholar Representative (D-Rep)? - %s", name),
			Description:        fmt.Sprintf("Will %s win the BITS Pilani Day Scholar Representative election?", name),
			OutcomeType:        "BINARY",
			ResolutionDateTime: resolutionTime,
			IsResolved:         false,
			InitialProbability: 0.5,
			YesLabel:           "YES",
			NoLabel:            "NO",
			LifecycleStatus:    "published",
			ApprovedBy:         "admin",
			ApprovedAt:         &now,
			CreatorUsername:    "admin",
			StewardUsername:    "admin",
		}
		if err := db.Create(&childMarket).Error; err != nil {
			return fmt.Errorf("creating drep child market %s: %w", name, err)
		}

		member := models.MarketGroupMember{
			GroupID:      drepGroup.ID,
			MarketID:     childMarket.ID,
			AnswerLabel:  name,
			DisplayOrder: i,
		}
		if err := db.Create(&member).Error; err != nil {
			return fmt.Errorf("creating drep market group member %s: %w", name, err)
		}
	}

	// Create H-Rep Groups for each Bhawan
	hrepCandidates := map[string][]string{
		"Ashok Bhawan":       {"Aryam Aman", "Kshitij Maheshwari"},
		"Bhagirath Bhawan":   {"Anurodh Sahu", "Ishan Garg", "Omkar Raju Thote"},
		"Budh Bhawan":        {"Ansh Agarwal"},
		"CVR Bhawan":         {"Arin Samant", "Shreyash Jha"},
		"Gandhi Bhawan":      {"Arsh Sood", "Kritanshu Jaiswal"},
		"Krishna Bhawan":     {"Anuj Miglani", "Daksh Singhal"},
		"Malviya Bhawan":     {"Ajay Sehrawat"},
		"Meera Bhawan":       {"Bhoomika Santosh Avadhani"},
		"Ram Bhawan":         {"Atrey Raj Singh", "Laksh Khetrapal", "Rudra Mohit"},
		"Rana Pratap Bhawan": {"Aryan Nair", "Nishque Sanodiya", "Shrey Dhorajiya"},
		"Shankar Bhawan":     {"Utsav Relan"},
		"Vishwakarma Bhawan": {"Anagh Kaushik", "Avi Dubey", "Priyanshu Satpathy"},
		"Vyas Bhawan":        {"Arnav Jain", "Jaideep Rankawat", "Manas Agrawal", "Raman Gupta", "Shashank Bansal", "Vinayak Raj"},
	}

	for bhawan, candidates := range hrepCandidates {
		bhawanGroup := models.MarketGroup{
			QuestionTitle:      fmt.Sprintf("Who will be elected H-Rep for %s?", bhawan),
			Description:        fmt.Sprintf("This market group resolves to the candidate who wins the BITS Pilani Hostel Representative (H-Rep) election for %s.", bhawan),
			GroupType:          "MULTIPLE_CHOICE_BINARY",
			ProbabilityPolicy:  "INDEPENDENT_BINARY",
			ResolutionPolicy:   "INDEPENDENT_CHILDREN",
			LifecycleStatus:    "published",
			ProposalCost:       0,
			CreatorUsername:    "admin",
			StewardUsername:    "admin",
			ApprovedBy:         "admin",
			ApprovedAt:         &now,
			ResolutionDateTime: resolutionTime,
		}

		if err := db.Create(&bhawanGroup).Error; err != nil {
			return fmt.Errorf("creating H-Rep market group for %s: %w", bhawan, err)
		}

		// Append "None of the Above"
		candidatesWithNone := append(candidates, "None of the Above")
		for i, name := range candidatesWithNone {
			childMarket := models.Market{
				QuestionTitle:      fmt.Sprintf("Who will be elected H-Rep for %s? - %s", bhawan, name),
				Description:        fmt.Sprintf("Will %s win the BITS Pilani H-Rep election for %s?", name, bhawan),
				OutcomeType:        "BINARY",
				ResolutionDateTime: resolutionTime,
				IsResolved:         false,
				InitialProbability: 0.5,
				YesLabel:           "YES",
				NoLabel:            "NO",
				LifecycleStatus:    "published",
				ApprovedBy:         "admin",
				ApprovedAt:         &now,
				CreatorUsername:    "admin",
				StewardUsername:    "admin",
			}
			if err := db.Create(&childMarket).Error; err != nil {
				return fmt.Errorf("creating H-Rep child market %s for %s: %w", name, bhawan, err)
			}

			member := models.MarketGroupMember{
				GroupID:      bhawanGroup.ID,
				MarketID:     childMarket.ID,
				AnswerLabel:  name,
				DisplayOrder: i,
			}
			if err := db.Create(&member).Error; err != nil {
				return fmt.Errorf("creating H-Rep market group member %s for %s: %w", name, bhawan, err)
			}
		}
	}

	logger.LogInfo("Seed", "SeedMarkets", "BITS Pilani election markets seeded successfully.")
	return nil
}

