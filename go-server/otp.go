package main

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OTP struct {
	Key     string
	Created time.Time
	Username string
}

type RetentionMap map[string]OTP

func NewRetentionMap(ctx context.Context, retentionPeriod time.Duration) RetentionMap{
	rm := make(RetentionMap)

	go rm.Retention(ctx, retentionPeriod)

	return rm
}

func (rm RetentionMap) NewOTP() OTP {
	o := OTP{
		Key: uuid.NewString(),
		Created: time.Now(),
	}

	rm[o.Key] = o
	return o
}

func (rm RetentionMap) VerifyOTP(otp string) bool{
	if _, ok := rm[otp]; !ok {
		return false // otp id not valid
	}
	delete(rm, otp)
	return true
}

func (rm RetentionMap) SetUsername(otpKey string, username string) {
	if otp, ok := rm[otpKey]; ok {
		otp.Username = username
		rm[otpKey] = otp
	}
}

func (rm RetentionMap) GetUsername(otpKey string) string {
	if otp, ok := rm[otpKey]; ok {
		return otp.Username
	}
	return ""
}

func (rm RetentionMap) Retention(ctx context.Context, retentionPeriod time.Duration){
	ticker := time.NewTicker(400 * time.Millisecond)

	for{
		select{
		case <- ticker.C:
			for _, otp := range rm{
				if otp.Created.Add(retentionPeriod).Before(time.Now()){
					delete(rm,otp.Key)
				}
			}
		case <-ctx.Done():
			return
		
		}
	}
}