package main

import (
	"net/http"
	"fmt"
)


func verifyAclValue(acl string, acls []string) bool {
	for _, v := range acls {
		if acl == v {
			return true
		}
	}

	return false
}

func s3VerifyAdmin(r *http.Request) error {
	access_token := r.Header.Get("X-SwyS3-Token")

	if access_token != s3Secrets[conf.Daemon.Token] {
		if S3ModeDevel {
			log.Errorf("Access token mismatch (%s!=%s)",
				access_token, s3Secrets[conf.Daemon.Token])
		}
		return fmt.Errorf("X-SwyS3-Token header mismatched or missed")
	}

	return nil
}

func s3CheckAccess(akey *S3AccessKey, bname, oname string) error {
	// FIXME Implement lookup and ACL, for now just allow
	return nil
}
