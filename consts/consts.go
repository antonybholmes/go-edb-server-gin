package consts

import (
	"crypto/rsa"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/antonybholmes/go-auth"
	"github.com/antonybholmes/go-sys/env"

	"github.com/golang-jwt/jwt/v5"
)

var NAME string
var APP_NAME string
var APP_URL string
var APP_DOMAIN string
var VERSION string
var COPYRIGHT string
var JWT_RSA_PRIVATE_KEY *rsa.PrivateKey //[]byte
var JWT_RSA_PUBLIC_KEY *rsa.PublicKey   //[]byte
var JWT_AUTH0_RSA_PUBLIC_KEY *rsa.PublicKey
var SESSION_NAME string
var SESSION_KEY string
var SESSION_ENCRYPTION_KEY string
var UPDATED string

var REDIS_ADDR string

var PASSWORDLESS_TOKEN_TTL_MINS time.Duration
var ACCESS_TOKEN_TTL_MINS time.Duration
var OTP_TOKEN_TTL_MINS time.Duration
var SHORT_TTL_MINS time.Duration

const DO_NOT_REPLY = "Please do not reply to this message. It was sent from a notification-only email address that we don't monitor."

func init() {
	env.Load("consts.env")
	env.Load("version.env")

	NAME = os.Getenv("NAME")
	APP_NAME = os.Getenv("APP_NAME")
	APP_URL = os.Getenv("APP_URL")
	APP_DOMAIN = os.Getenv("APP_DOMAIN")
	VERSION = os.Getenv("VERSION")
	UPDATED = os.Getenv("UPDATED")
	COPYRIGHT = os.Getenv("COPYRIGHT")

	REDIS_ADDR = os.Getenv("REDIS_ADDR")

	//JWT_PRIVATE_KEY = []byte(os.Getenv("JWT_SECRET"))
	//JWT_PUBLIC_KEY = []byte(os.Getenv("JWT_SECRET"))
	SESSION_NAME = os.Getenv("SESSION_NAME")
	SESSION_KEY = os.Getenv("SESSION_KEY")
	SESSION_ENCRYPTION_KEY = os.Getenv("SESSION_ENCRYPTION_KEY")

	PASSWORDLESS_TOKEN_TTL_MINS = env.GetMin("PASSWORDLESS_TOKEN_TTL_MINS", auth.TTL_10_MINS)
	ACCESS_TOKEN_TTL_MINS = env.GetMin("ACCESS_TOKEN_TTL_MINS", auth.TTL_15_MINS)
	OTP_TOKEN_TTL_MINS = env.GetMin("OTP_TOKEN_TTL_MINS", auth.TTL_20_MINS)
	SHORT_TTL_MINS = env.GetMin("SHORT_TTL_MINS", auth.TTL_10_MINS)

	bytes, err := os.ReadFile("jwtRS256.key")
	if err != nil {
		log.Fatal().Msgf("%s", err)
	}

	JWT_RSA_PRIVATE_KEY, err = jwt.ParseRSAPrivateKeyFromPEM(bytes)
	if err != nil {
		log.Fatal().Msgf("%s", err)
	}

	bytes, err = os.ReadFile("jwtRS256.key.pub")
	if err != nil {
		log.Fatal().Msgf("%s", err)
	}

	JWT_RSA_PUBLIC_KEY, err = jwt.ParseRSAPublicKeyFromPEM(bytes)
	if err != nil {
		log.Fatal().Msgf("%s", err)
	}

	bytes, err = os.ReadFile("auth0.key.pub")
	if err != nil {
		log.Fatal().Msgf("%s", err)
	}

	JWT_AUTH0_RSA_PUBLIC_KEY, err = jwt.ParseRSAPublicKeyFromPEM(bytes)
	if err != nil {
		log.Fatal().Msgf("%s", err)
	}

}
