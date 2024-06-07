package utils

import "golang.org/x/crypto/bcrypt"

func PasswordHash(password string) string {
	passwordHashBytes, _ := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(passwordHashBytes)
}

func CheckPasswordHash(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
