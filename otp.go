package main

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OTP struct {
	Key     string
	Created time.Time
}

type OTPList map[string]OTP

func NewOTPList(ctx context.Context, ttl time.Duration) OTPList {
	otpList := make(OTPList)
	go otpList.CleanUp(ctx, ttl)
	return otpList
}

func (otpList OTPList) NewOTP() OTP {
	otp := OTP{
		Key:     uuid.New().String(),
		Created: time.Now(),
	}
	otpList[otp.Key] = otp
	return otp
}

func (otpList OTPList) ValidateOTP(key string) bool {
	if _, ok := otpList[key]; !ok {
		return false
	}
	delete(otpList, key)
	return true
}

func (otpList OTPList) CleanUp(ctx context.Context, ttl time.Duration) {
	ticker := time.NewTicker(400 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			for key, otp := range otpList {
				if otp.Created.Add(ttl).Before(time.Now()) {
					delete(otpList, key)
				}
			}
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}
