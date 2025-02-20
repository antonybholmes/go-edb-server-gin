package authorization

import (
	"github.com/antonybholmes/go-auth"
	"github.com/antonybholmes/go-auth/tokengen"
	"github.com/antonybholmes/go-edb-server-gin/consts"
	"github.com/antonybholmes/go-edb-server-gin/routes"
	authenticationroutes "github.com/antonybholmes/go-edb-server-gin/routes/authentication"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthReq struct {
	Authorization string `header:"Authorization"`
}

// func RenewTokenRoute(c *gin.Context) {
// 	user := c.Get("user").(*jwt.Token)
// 	claims := user.Claims.(*auth.JwtCustomClaims)

// 	// Throws unauthorized error
// 	//if username != "edb" || password != "tod4EwVHEyCRK8encuLE" {
// 	//	return echo.ErrUnauthorized
// 	//}

// 	// Set custom claims
// 	renewClaims := auth.JwtCustomClaims{
// 		UserId: claims.UserId,
// 		//Email: authUser.Email,
// 		IpAddr: claims.IpAddr,
// 		RegisteredClaims: jwt.RegisteredClaims{
// 			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * auth.JWT_TOKEN_EXPIRES_HOURS)),
// 		},
// 	}

// 	// Create token with claims
// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, renewClaims)

// 	// Generate encoded token and send it as response.
// 	t, err := token.SignedString([]byte(consts.JWT_SECRET))

// 	if err != nil {
// 		return routes.ErrorReq("error signing token")
// 	}

// 	return MakeDataResp(c, "", &JwtResp{t})
// }

func TokenInfoRoute(c *gin.Context) {
	t, err := routes.HeaderAuthToken(c)

	if err != nil {
		c.Error(err)
		return
	}

	claims := auth.TokenClaims{}

	_, err = jwt.ParseWithClaims(t, &claims, func(token *jwt.Token) (interface{}, error) {
		return consts.JWT_RSA_PUBLIC_KEY, nil
	})

	if err != nil {
		c.Error(err)
		return
	}

	routes.MakeDataResp(c, "", &routes.JwtInfo{
		Uuid: claims.UserId,
		Type: claims.Type, //.TokenTypeString(claims.Type),
		//IpAddr:  claims.IpAddr,
		Expires: claims.ExpiresAt.UTC().String()})

}

func NewAccessTokenRoute(c *gin.Context) {
	authenticationroutes.NewValidator(c).CheckIsValidRefreshToken().Success(func(validator *authenticationroutes.Validator) {

		// Generate encoded token and send it as response.
		accessToken, err := tokengen.AccessToken(c, validator.Claims.UserId, validator.Claims.Roles)

		if err != nil {
			routes.ErrorResp(c, "error creating access token")
		}

		routes.MakeDataResp(c, "", &routes.AccessTokenResp{AccessToken: accessToken})
	})

}
