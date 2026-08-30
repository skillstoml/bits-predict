package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"socialpredict/handlers"
	"socialpredict/models"
	configsvc "socialpredict/internal/service/config"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type signupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func SignupHandler(db *gorm.DB, configService configsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			handlers.WriteFailure(w, http.StatusMethodNotAllowed, handlers.ReasonMethodNotAllowed)
			return
		}

		var req signupRequest
		decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
		if err := decoder.Decode(&req); err != nil {
			handlers.WriteFailure(w, http.StatusBadRequest, handlers.ReasonInvalidRequest)
			return
		}

		req.Username = strings.TrimSpace(strings.ToLower(req.Username))
		req.Email = strings.TrimSpace(strings.ToLower(req.Email))

		// 1. Validate Username (alphanumeric, min 3, max 30)
		usernameRegex := regexp.MustCompile(`^[a-z0-9_]{3,30}$`)
		if !usernameRegex.MatchString(req.Username) {
			handlers.WriteFailure(w, http.StatusBadRequest, handlers.FailureReason("Username must be 3-30 characters and alphanumeric/underscore"))
			return
		}

		// 2. Validate Password (min 6)
		if len(req.Password) < 6 {
			handlers.WriteFailure(w, http.StatusBadRequest, handlers.FailureReason("Password must be at least 6 characters"))
			return
		}

		// 3. Validate Email Domain
		// Check that email has @ and has correct suffix
		parts := strings.Split(req.Email, "@")
		if len(parts) != 2 || (!strings.HasSuffix(parts[1], "bits-pilani.ac.in") && parts[1] != "bits-pilani.ac.in") {
			handlers.WriteFailure(w, http.StatusBadRequest, handlers.FailureReason("Only BITS Pilani email addresses are allowed (@*bits-pilani.ac.in)"))
			return
		}

		// 4. Check if username already exists
		var count int64
		if err := db.Model(&models.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
			handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}
		if count > 0 {
			handlers.WriteFailure(w, http.StatusConflict, handlers.FailureReason("Username is already taken"))
			return
		}

		// 5. Check if email already exists
		if err := db.Model(&models.User{}).Where("email = ?", req.Email).Count(&count).Error; err != nil {
			handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}
		if count > 0 {
			handlers.WriteFailure(w, http.StatusConflict, handlers.FailureReason("Email is already registered"))
			return
		}

		// 6. Create User GORM model
		userConfig := configService.Economics().User
		user := models.User{
			PublicUser: models.PublicUser{
				Username:              req.Username,
				DisplayName:           req.Username, // Default display name to username
				UserType:              "REGULAR",
				InitialAccountBalance: userConfig.InitialAccountBalance,
				AccountBalance:        userConfig.InitialAccountBalance,
				PersonalEmoji:         "NONE",
			},
			PrivateUser: models.PrivateUser{
				Email:  req.Email,
				APIKey: uuid.NewString(),
			},
			MustChangePassword: false, // Self-created users set their password at signup
		}

		if err := user.HashPassword(req.Password); err != nil {
			handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}

		if err := db.Create(&user).Error; err != nil {
			handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}

		handlers.WriteResult(w, http.StatusCreated, map[string]string{
			"status":   "success",
			"username": user.Username,
		})
	}
}
