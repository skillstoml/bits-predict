package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"socialpredict/handlers"
	"socialpredict/models"
	configsvc "socialpredict/internal/service/config"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"
)

type googleLoginRequest struct {
	Credential string `json:"credential"`
}

func GoogleLoginHandler(db *gorm.DB, configService configsvc.Service, jwtSigningKey []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			handlers.WriteFailure(w, http.StatusMethodNotAllowed, handlers.ReasonMethodNotAllowed)
			return
		}

		var req googleLoginRequest
		decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
		if err := decoder.Decode(&req); err != nil {
			handlers.WriteFailure(w, http.StatusBadRequest, handlers.ReasonInvalidRequest)
			return
		}

		if req.Credential == "" {
			handlers.WriteFailure(w, http.StatusBadRequest, handlers.FailureReason("Google credential token is missing"))
			return
		}

		googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
		if googleClientID == "" {
			handlers.WriteFailure(w, http.StatusInternalServerError, handlers.FailureReason("Google Sign-In is not configured on the server"))
			return
		}

		// 1. Verify Google ID token
		payload, err := idtoken.Validate(r.Context(), req.Credential, googleClientID)
		if err != nil {
			handlers.WriteFailure(w, http.StatusUnauthorized, handlers.FailureReason(fmt.Sprintf("Invalid Google token: %v", err)))
			return
		}

		// 2. Extract Claims
		email, ok := payload.Claims["email"].(string)
		if !ok || email == "" {
			handlers.WriteFailure(w, http.StatusBadRequest, handlers.FailureReason("Email missing from Google token"))
			return
		}
		email = strings.TrimSpace(strings.ToLower(email))

		emailVerified, ok := payload.Claims["email_verified"].(bool)
		if !ok || !emailVerified {
			handlers.WriteFailure(w, http.StatusUnauthorized, handlers.FailureReason("Google email address is not verified"))
			return
		}

		// 3. Validate Email Domain
		parts := strings.Split(email, "@")
		if len(parts) != 2 || (!strings.HasSuffix(parts[1], "bits-pilani.ac.in") && parts[1] != "bits-pilani.ac.in") {
			handlers.WriteFailure(w, http.StatusBadRequest, handlers.FailureReason("Only BITS Pilani email addresses are allowed (@*bits-pilani.ac.in)"))
			return
		}

		// 4. Check if user already exists
		var user models.User
		err = db.Where("email = ?", email).First(&user).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// User doesn't exist, register them automatically
				baseUsername := strings.Split(email, "@")[0]
				baseUsername = strings.ReplaceAll(baseUsername, ".", "_")
				baseUsername = strings.ReplaceAll(baseUsername, "-", "_")

				// Ensure username matches alphanumeric regex
				usernameRegex := regexp.MustCompile(`^[a-z0-9_]{3,30}$`)
				if !usernameRegex.MatchString(baseUsername) {
					// Fallback to random string if prefix is invalid
					baseUsername = "user_" + uuid.NewString()[:8]
				}

				// Check username uniqueness, append random suffix if taken
				username := baseUsername
				for {
					var count int64
					if err := db.Model(&models.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
						handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
						return
					}
					if count == 0 {
						break
					}
					username = fmt.Sprintf("%s_%s", baseUsername, uuid.NewString()[:4])
				}

				userConfig := configService.Economics().User
				user = models.User{
					PublicUser: models.PublicUser{
						Username:              username,
						DisplayName:           username,
						UserType:              "REGULAR",
						InitialAccountBalance: userConfig.InitialAccountBalance,
						AccountBalance:        userConfig.InitialAccountBalance,
						PersonalEmoji:         "NONE",
					},
					PrivateUser: models.PrivateUser{
						Email:  email,
						APIKey: uuid.NewString(),
						// Set random password since they login via Google OAuth
						Password: uuid.NewString(),
					},
					MustChangePassword: false, // Login via Google doesn't need password reset
				}

				// Hash the fake password
				if err := user.HashPassword(user.Password); err != nil {
					handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
					return
				}

				if err := db.Create(&user).Error; err != nil {
					handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
					return
				}
			} else {
				handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
				return
			}
		}

		// 5. Generate session token
		tokenString, err := generateJWT(user.Username, jwtSigningKey)
		if err != nil {
			handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}

		// 6. Write response matching login response format
		handlers.WriteResult(w, http.StatusOK, loginResponse{
			Token:              tokenString,
			Username:           user.Username,
			UserType:           user.UserType,
			ModeratorStatus:    user.ModeratorStatus,
			MustChangePassword: user.MustChangePassword,
		})
	}
}
