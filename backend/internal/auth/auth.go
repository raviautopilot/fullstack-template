package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"backend/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Claims struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	jwt.RegisteredClaims
}

var googleOauthConfig *oauth2.Config

func InitOauth() {
	if config.ActiveConfig == nil {
		return
	}
	googleOauthConfig = &oauth2.Config{
		ClientID:     config.ActiveConfig.GoogleClientID,
		ClientSecret: config.ActiveConfig.GoogleClientSecret,
		RedirectURL:  config.ActiveConfig.OAuthRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}
}

// LoginHandler godoc
// @Summary      Initiate Google OAuth Login
// @Description  Redirects the user to Google OAuth Consent screen, or a Mock OAuth consent screen when OAUTH_MODE=mock
// @Tags         Auth
// @Produce      html
// @Success      302  {string}  string  "Redirect to Google or Mock portal"
// @Router       /auth/login [get]
func LoginHandler(c *gin.Context) {
	state := c.Query("state")
	if state == "" {
		state = "random-auth-state-token"
	}

	if config.ActiveConfig.OAuthMode == "mock" {
		// Redirect to our local mock consent screen
		mockConsentURL := fmt.Sprintf("/auth/mock-consent?state=%s", state)
		c.Redirect(http.StatusTemporaryRedirect, mockConsentURL)
		return
	}

	// Real Google OAuth redirect
	url := googleOauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// MockConsentHandler renders a premium mock Google OAuth consent portal
func MockConsentHandler(c *gin.Context) {
	state := c.Query("state")
	
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Google Accounts - Sign in</title>
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600&display=swap" rel="stylesheet">
    <style>
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
            font-family: 'Outfit', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
        }
        body {
            background-color: #0b0f19;
            color: #f3f4f6;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            overflow: hidden;
        }
        .container {
            width: 450px;
            padding: 40px;
            background: rgba(17, 24, 39, 0.7);
            border: 1px solid rgba(255, 255, 255, 0.08);
            border-radius: 24px;
            backdrop-filter: blur(16px);
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4);
            text-align: center;
            position: relative;
        }
        .container::before {
            content: '';
            position: absolute;
            top: -2px; left: -2px; right: -2px; bottom: -2px;
            background: linear-gradient(135deg, #3b82f6, #8b5cf6);
            border-radius: 24px;
            z-index: -1;
            opacity: 0.15;
        }
        .logo-box {
            display: flex;
            justify-content: center;
            align-items: center;
            gap: 8px;
            margin-bottom: 24px;
        }
        .logo-dot {
            width: 12px;
            height: 12px;
            border-radius: 50%%;
        }
        .dot-red { background-color: #ea4335; }
        .dot-blue { background-color: #4285f4; }
        .dot-yellow { background-color: #fbbc05; }
        .dot-green { background-color: #34a853; }
        
        .title {
            font-size: 24px;
            font-weight: 600;
            margin-bottom: 8px;
            background: linear-gradient(to right, #60a5fa, #a78bfa);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .subtitle {
            font-size: 14px;
            color: #9ca3af;
            margin-bottom: 32px;
        }
        .dev-badge {
            background-color: rgba(59, 130, 246, 0.1);
            color: #60a5fa;
            border: 1px solid rgba(59, 130, 246, 0.2);
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 12px;
            display: inline-block;
            margin-bottom: 24px;
            font-weight: 500;
        }
        .profile-list {
            display: flex;
            flex-direction: column;
            gap: 12px;
            margin-bottom: 24px;
        }
        .profile-btn {
            display: flex;
            align-items: center;
            width: 100%%;
            padding: 14px 20px;
            background: rgba(255, 255, 255, 0.03);
            border: 1px solid rgba(255, 255, 255, 0.06);
            border-radius: 16px;
            color: #f3f4f6;
            cursor: pointer;
            text-align: left;
            transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
        }
        .profile-btn:hover {
            background: rgba(255, 255, 255, 0.08);
            border-color: rgba(59, 130, 246, 0.4);
            transform: translateY(-2px);
        }
        .avatar {
            width: 40px;
            height: 40px;
            border-radius: 50%%;
            margin-right: 16px;
            display: flex;
            justify-content: center;
            align-items: center;
            font-weight: 600;
            font-size: 18px;
            color: white;
        }
        .av-1 { background: linear-gradient(135deg, #ef4444, #f97316); }
        .av-2 { background: linear-gradient(135deg, #3b82f6, #06b6d4); }
        .av-3 { background: linear-gradient(135deg, #10b981, #14b8a6); }
        
        .profile-info {
            flex-grow: 1;
        }
        .profile-name {
            font-size: 15px;
            font-weight: 500;
        }
        .profile-email {
            font-size: 13px;
            color: #9ca3af;
        }
        .custom-form {
            border-top: 1px solid rgba(255, 255, 255, 0.08);
            padding-top: 20px;
            margin-top: 16px;
            text-align: left;
        }
        .form-title {
            font-size: 14px;
            font-weight: 500;
            margin-bottom: 12px;
            color: #9ca3af;
        }
        .input-group {
            display: flex;
            flex-direction: column;
            gap: 6px;
            margin-bottom: 12px;
        }
        label {
            font-size: 12px;
            color: #6b7280;
        }
        input {
            width: 100%%;
            padding: 10px 14px;
            background: rgba(0, 0, 0, 0.2);
            border: 1px solid rgba(255, 255, 255, 0.1);
            border-radius: 8px;
            color: white;
            font-size: 14px;
            outline: none;
            transition: border-color 0.2s;
        }
        input:focus {
            border-color: #60a5fa;
        }
        .submit-btn {
            width: 100%%;
            padding: 12px;
            background: linear-gradient(135deg, #3b82f6, #8b5cf6);
            border: none;
            border-radius: 12px;
            color: white;
            font-weight: 600;
            cursor: pointer;
            transition: opacity 0.2s;
            margin-top: 8px;
        }
        .submit-btn:hover {
            opacity: 0.9;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo-box">
            <div class="logo-dot dot-blue"></div>
            <div class="logo-dot dot-red"></div>
            <div class="logo-dot dot-yellow"></div>
            <div class="logo-dot dot-green"></div>
        </div>
        <h1 class="title">Mock OAuth Portal</h1>
        <p class="subtitle">Secure Testing Consent Environment</p>
        
        <span class="dev-badge">Local & E2E Testing Enabled</span>
        
        <div class="profile-list">
            <button class="profile-btn" data-testid="mock-login-user-btn" onclick="selectProfile('Test Developer', 'test_dev@example.com', 'https://avatar.iran.liara.run/public/31')">
                <div class="avatar av-1">D</div>
                <div class="profile-info">
                    <div class="profile-name">Test Developer</div>
                    <div class="profile-email">test_dev@example.com</div>
                </div>
            </button>
            <button class="profile-btn" onclick="selectProfile('QA Engineer', 'qa_tester@example.com', 'https://avatar.iran.liara.run/public/60')">
                <div class="avatar av-2">Q</div>
                <div class="profile-info">
                    <div class="profile-name">QA Engineer</div>
                    <div class="profile-email">qa_tester@example.com</div>
                </div>
            </button>
        </div>

        <form class="custom-form" onsubmit="submitCustom(event)">
            <p class="form-title">Or use a custom profile</p>
            <div class="input-group">
                <label for="custom-name">Full Name</label>
                <input type="text" id="custom-name" placeholder="John Doe" required>
            </div>
            <div class="input-group">
                <label for="custom-email">Email Address</label>
                <input type="email" id="custom-email" placeholder="john@example.com" required>
            </div>
            <button type="submit" class="submit-btn">Authorize Custom Account</button>
        </form>
    </div>

    <script>
        function selectProfile(name, email, pic) {
            const state = "%s";
            const code = "MOCK_CODE_" + btoa(JSON.stringify({name, email, pic}));
            window.location.href = "/auth/callback?code=" + encodeURIComponent(code) + "&state=" + encodeURIComponent(state);
        }
        function submitCustom(e) {
            e.preventDefault();
            const name = document.getElementById('custom-name').value;
            const email = document.getElementById('custom-email').value;
            const pic = "https://avatar.iran.liara.run/public/user";
            selectProfile(name, email, pic);
        }
    </script>
</body>
</html>`, state)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
}

// CallbackHandler godoc
// @Summary      Google OAuth Callback Endpoint
// @Description  Exchanges authorization code for access token and logs in user, setting a JWT token. Redirects to frontend.
// @Tags         Auth
// @Produce      json
// @Param        code   query      string  true  "Authorization Code"
// @Param        state  query      string  true  "OAuth State"
// @Success      302    {string}   string  "Redirect back to react app with token"
// @Router       /auth/callback [get]
func CallbackHandler(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	_ = state // Can validate state here if necessary

	var name, email, picture string

	// Handle Mock Callback
	if config.ActiveConfig.OAuthMode == "mock" && strings.HasPrefix(code, "MOCK_CODE_") {
		encodedStr := strings.TrimPrefix(code, "MOCK_CODE_")
		// standard base64 decoding
		
		type MockUser struct {
			Name    string `json:"name"`
			Email   string `json:"email"`
			Picture string `json:"pic"`
		}
		var mu MockUser
		// Decode standard base64 manually or use json direct if not base64. 
		// Actually, in our javascript we btoa-encoded it: "MOCK_CODE_" + btoa(...)
		// Let's decode it safely:
		importString := strings.ReplaceAll(encodedStr, " ", "+") // fix any URL encoding gaps
		decodedBytes, err := base64Decode(importString)
		if err == nil {
			_ = json.Unmarshal(decodedBytes, &mu)
		} else {
			// fallback if parsing failed
			mu.Name = "Mock User"
			mu.Email = "mock_user@example.com"
			mu.Picture = ""
		}

		name = mu.Name
		email = mu.Email
		picture = mu.Picture
	} else {
		// Real Google OAuth callback flow
		if googleOauthConfig == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth is not initialized"})
			return
		}
		token, err := googleOauthConfig.Exchange(context.Background(), code)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to exchange code: %v", err)})
			return
		}

		client := googleOauthConfig.Client(context.Background(), token)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to get userinfo: %v", err)})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read userinfo response"})
			return
		}

		var userInfo map[string]interface{}
		if err := json.Unmarshal(body, &userInfo); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse userinfo"})
			return
		}

		if val, ok := userInfo["name"].(string); ok {
			name = val
		}
		if val, ok := userInfo["email"].(string); ok {
			email = val
		}
		if val, ok := userInfo["picture"].(string); ok {
			picture = val
		}
	}

	// Create JWT token
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Email:   email,
		Name:    name,
		Picture: picture,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.ActiveConfig.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session token"})
		return
	}

	// Set cookie with token for server sessions (Secure/HttpOnly if environment allows)
	// For testing, we also put it in the URL so frontend can easily store it in localStorage.
	c.SetCookie("auth_token", tokenString, 3600*24, "/", "", false, false)

	// Redirect back to frontend with the token
	redirectURL := fmt.Sprintf("%s/?token=%s", config.ActiveConfig.FrontendURL, tokenString)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// UserHandler godoc
// @Summary      Get Logged In User Profile
// @Description  Validates the JWT token in authorization headers or cookie and returns user profile details
// @Tags         Auth
// @Produce      json
// @Param        Authorization  header    string  false  "Bearer Token"
// @Success      200            {object}  map[string]string
// @Failure      401            {object}  map[string]string
// @Router       /api/user [get]
func UserHandler(c *gin.Context) {
	var tokenStr string
	
	// Check Authorization Header
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Fallback to Cookie
	if tokenStr == "" {
		cookieToken, err := c.Cookie("auth_token")
		if err == nil {
			tokenStr = cookieToken
		}
	}

	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized. Missing token."})
		return
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.ActiveConfig.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized. Invalid token."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email":   claims.Email,
		"name":    claims.Name,
		"picture": claims.Picture,
	})
}

// LogoutHandler godoc
// @Summary      Logout User
// @Description  Clears session cookies and logs the user out
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /auth/logout [get]
func LogoutHandler(c *gin.Context) {
	// Clear the auth_token cookie
	c.SetCookie("auth_token", "", -1, "/", "", false, false)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// Helper base64 decoder
func base64Decode(str string) ([]byte, error) {
	// Standard padding replacement
	str = strings.ReplaceAll(str, "-", "+")
	str = strings.ReplaceAll(str, "_", "/")
	
	switch len(str) % 4 {
	case 2:
		str += "=="
	case 3:
		str += "="
	}

	// Using standard base64 decoding helper
	var dec []byte
	_, err := fmt.Sscanf(str, "%%s", &dec) // Dummy to bypass direct compile if we don't import
	// Standard base64 decoding implementation:
	importDec, err := b64decode(str)
	return importDec, err
}

func b64decode(s string) ([]byte, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var lookup [256]byte
	for i := 0; i < len(alphabet); i++ {
		lookup[alphabet[i]] = byte(i)
	}

	s = strings.TrimRight(s, "=")
	n := len(s)
	if n == 0 {
		return nil, nil
	}

	dst := make([]byte, (n*6)/8)
	var val uint32
	var bits int
	var idx int

	for i := 0; i < n; i++ {
		ch := s[i]
		if ch == '+' {
			ch = '+'
		} else if ch == '/' {
			ch = '/'
		}
		
		var val6 byte
		if lookup[ch] == 0 && ch != 'A' {
			// error case
			continue
		}
		val6 = lookup[ch]

		val = (val << 6) | uint32(val6)
		bits += 6
		if bits >= 8 {
			bits -= 8
			dst[idx] = byte(val >> bits)
			idx++
		}
	}
	return dst[:idx], nil
}
