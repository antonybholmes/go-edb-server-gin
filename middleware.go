package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/antonybholmes/go-auth"
	"github.com/antonybholmes/go-edb-server-gin/consts"
	"github.com/antonybholmes/go-edb-server-gin/routes"
	authenticationroutes "github.com/antonybholmes/go-edb-server-gin/routes/authentication"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/golang-jwt/jwt/v5"
)

type APIError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func LoggingMiddleware(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process the request
		c.Next()

		// Log the HTTP request details after it completes
		duration := time.Since(start)

		// Log the request information
		logger.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("duration", duration).
			Msg("HTTP request")
	}
}

func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a defer function that will be called after the handler finishes
		defer func() {
			if err := recover(); err != nil {
				// Handle panic errors with 500 status code
				c.JSON(http.StatusInternalServerError, APIError{
					Code:    http.StatusInternalServerError,
					Message: fmt.Sprintf("Internal Server Error: %v", err),
				})
			}
		}()

		// Continue processing the request
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			// Get the last error (or you can choose how to handle multiple errors)
			err := c.Errors.Last()

			// Set a custom status code based on the error
			// If no custom status code is set, use the error's default status or fallback to 400
			statusCode := http.StatusBadRequest

			if err.Meta != nil {
				// ok indicates cast worked
				customStatus, ok := err.Meta.(int)

				if ok {
					statusCode = customStatus
				}
			}

			// Send the error response with custom status code
			c.JSON(statusCode, APIError{
				Code:    statusCode,
				Message: err.Error(),
			})
		}
	}
}

func parseToken(c *gin.Context) (string, error) {
	// Get the token from the "Authorization" header
	authHeader := c.GetHeader("Authorization")

	if authHeader == "" {
		return "", fmt.Errorf("authorization header missing")
	}

	// Split the token (format: "Bearer <token>")
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	if tokenString == authHeader {
		return "", fmt.Errorf("malformed token")
	}

	return tokenString, nil
}

func JwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		tokenString, err := parseToken(c)

		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		claims := auth.TokenClaims{}

		// Parse the JWT
		_, err = jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
			// Return the secret key for verifying the token
			return consts.JWT_RSA_PUBLIC_KEY, nil
		})

		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		// use pointer to token
		c.Set("user", &claims)

		// Continue processing the request
		c.Next()

	}
}

func JwtAuth0Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		tokenString, err := parseToken(c)

		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		claims := auth.Auth0TokenClaims{}

		log.Debug().Msgf("tok %s", tokenString)

		// Parse the JWT
		_, err = jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
			// Return the secret key for verifying the token
			return consts.JWT_AUTH0_RSA_PUBLIC_KEY, nil
		})

		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		// use pointer to token
		c.Set("user", &claims)

		// Continue processing the request
		c.Next()
	}
}

type UserClaimsFunc func(c *gin.Context, claims *auth.TokenClaims)

func checkUserExistsMiddleware(c *gin.Context, f UserClaimsFunc) {

	user, ok := c.Get("user")

	if !ok {
		routes.UserDoesNotExistResp(c)

		return
	}

	claims := user.(*auth.TokenClaims)

	f(c, claims)
}

func JwtSpecificTokenMiddleware(tokenType auth.TokenType) gin.HandlerFunc {
	return func(c *gin.Context) {
		checkUserExistsMiddleware(c, func(c *gin.Context, claims *auth.TokenClaims) {

			if claims.Type != tokenType {
				routes.AuthErrorResp(c, fmt.Sprintf("wrong token type: %s, should be %s", claims.Type, tokenType))

				return
			}

			c.Next()
		})
	}
}

func JwtIsRefreshTokenMiddleware() gin.HandlerFunc {
	return JwtSpecificTokenMiddleware(auth.REFRESH_TOKEN)
}

func JwtIsAccessTokenMiddleware() gin.HandlerFunc {
	return JwtSpecificTokenMiddleware(auth.ACCESS_TOKEN)
}

func JwtIsVerifyEmailTokenMiddleware() gin.HandlerFunc {
	return JwtSpecificTokenMiddleware(auth.VERIFY_EMAIL_TOKEN)
}

func JwtIsAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		checkUserExistsMiddleware(c, func(c *gin.Context, claims *auth.TokenClaims) {

			if !auth.IsAdmin((claims.Roles)) {
				routes.AuthErrorResp(c, "user is not an admin")

				return
			}

			c.Next()
		})
	}
}

func JwtCanSigninMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		checkUserExistsMiddleware(c, func(c *gin.Context, claims *auth.TokenClaims) {

			if !auth.CanSignin((claims.Roles)) {
				routes.AuthErrorResp(c, "user is not allowed to login")
				return
			}

			c.Next()
		})
	}
}

// basic check that session exists and seems to be populated with the user
func SessionIsValidMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessData, err := authenticationroutes.ReadSessionInfo(c)

		if err != nil {
			routes.AuthErrorResp(c, "cannot get user id from session")

			return
		}

		c.Set("authUser", sessData.AuthUser)

		c.Next()
	}
}

// func ValidateJwtMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
// 	return func(c *gin.Context) {
// 		authorizationHeader := c.Request().Header.Get("authorization")

// 		if len(authorizationHeader) == 0 {
// 			return routes.AuthErrorReq("missing Authentication header")

// 		}

// 		log.Debug().Msgf("parsing authentication header")

// 		authPair := strings.SplitN(authorizationHeader, " ", 2)

// 		if len(authPair) != 2 {
// 			return routes.AuthErrorReq("wrong Authentication header definiton")
// 		}

// 		headerAuthScheme := authPair[0]
// 		headerAuthToken := authPair[1]

// 		if headerAuthScheme != "Bearer" {
// 			return routes.AuthErrorReq("wrong Authentication header definiton")
// 		}

// 		log.Debug().Msgf("validating JWT token")

// 		token, err := validateJwtToken(headerAuthToken)

// 		if err != nil {
// 			return routes.AuthErrorReq(err)
// 		}

// 		log.Debug().Msgf("JWT token is valid")
// 		c.Set("user", token)
// 		return next(c)

// 	}
// }

// Create a permissions middleware to verify jwt permissions on a token
func JwtRoleMiddleware(validRoles ...string) gin.HandlerFunc {

	return func(c *gin.Context) {

		checkUserExistsMiddleware(c, func(c *gin.Context, claims *auth.TokenClaims) {

			//log.Debug().Msgf("claims %v", claims)

			// if we are not an admin, lets see what roles
			// we have and if they match the valid list
			if !auth.IsAdmin(claims.Roles) {
				isValidRole := false

				for _, validRole := range validRoles {

					// if we find a permission, stop and move on
					if strings.Contains(claims.Roles, validRole) {
						isValidRole = true
						break
					}

				}

				if !isValidRole {
					routes.ErrorResp(c, "invalid role")
					return
				}
			}

			c.Next()
		})
	}
}

func RDFMiddleware() gin.HandlerFunc {
	return JwtRoleMiddleware("RDF")
}
