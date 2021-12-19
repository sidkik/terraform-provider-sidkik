package sidkik

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccFirebaseAuthConfig_config(t *testing.T) {
	t.Parallel()

	context := map[string]interface{}{}

	vcrTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFirebaseAuthConfigDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				Config: testAccFirebaseAuthConfig_config(context),
			},
			{
				ResourceName:      "sidkik_firebase_auth_config.config",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccFirebaseAuthConfig_config(context map[string]interface{}) string {
	return Nprintf(`
resource "sidkik_firebase_auth_config" "config" {
	email = true
	authorized_domains =["my-account.sidkik.app", "admin-my-account.sidkik.app"]
}
`, context)
}

func testAccCheckFirebaseAuthConfigDestroyProducer(t *testing.T) func(s *terraform.State) error {
	return func(s *terraform.State) error {
		// for name, rs := range s.RootModule().Resources {
		// 	if rs.Type != "sidkik_firebase" {
		// 		continue
		// 	}
		// 	if strings.HasPrefix(name, "data.") {
		// 		continue
		// 	}

		// 	config := googleProviderConfig(t)

		// 	url, err := replaceVarsForTest(config, rs, "{{FirebaseRulesBasePath}}projects/{{project}}/rulesets")
		// 	if err != nil {
		// 		return err
		// 	}

		// 	project := ""

		// 	if config.Project != "" {
		// 		project = config.Project
		// 	}

		// 	_, err = sendRequest(config, "GET", project, url, config.userAgent, nil)
		// 	if err == nil {
		// 		return fmt.Errorf("ComputeHealthCheck still exists at %s", url)
		// 	}
		// }

		return nil
	}
}

func Test_flattenEmail(t *testing.T) {
	type args struct {
		response string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "populated",
			args: args{
				response: `{
					"name": "projects/125051500756/config",
					"signIn": {
						"email": {
							"enabled": true,
							"passwordRequired": true
						},
						"hashConfig": {
							"algorithm": "SCRYPT",
							"signerKey": "Rqol0YtxtvFoOn28eAg4/W7UkIfrBypR5mmsZy7e1JydqX8M3zm2oc+osHn9j4yh9fKO/27KhlJm25mgBg5Ebw==",
							"saltSeparator": "Bw==",
							"rounds": 8,
							"memoryCost": 14
						}
					},
					"notification": {
						"sendEmail": {
							"method": "DEFAULT",
							"resetPasswordTemplate": {
								"senderLocalPart": "noreply",
								"subject": "Reset your password for %APP_NAME%",
								"body": "\u003cp\u003eHello,\u003c/p\u003e\n\u003cp\u003eFollow this link to reset your %APP_NAME% password for your %EMAIL% account.\u003c/p\u003e\n\u003cp\u003e\u003ca href='%LINK%'\u003e%LINK%\u003c/a\u003e\u003c/p\u003e\n\u003cp\u003eIf you didn’t ask to reset your password, you can ignore this email.\u003c/p\u003e\n\u003cp\u003eThanks,\u003c/p\u003e\n\u003cp\u003eYour %APP_NAME% team\u003c/p\u003e",
								"bodyFormat": "HTML",
								"replyTo": "noreply"
							},
							"verifyEmailTemplate": {
								"senderLocalPart": "noreply",
								"subject": "Verify your email for %APP_NAME%",
								"body": "\u003cp\u003eHello %DISPLAY_NAME%,\u003c/p\u003e\n\u003cp\u003eFollow this link to verify your email address.\u003c/p\u003e\n\u003cp\u003e\u003ca href='%LINK%'\u003e%LINK%\u003c/a\u003e\u003c/p\u003e\n\u003cp\u003eIf you didn’t ask to verify this address, you can ignore this email.\u003c/p\u003e\n\u003cp\u003eThanks,\u003c/p\u003e\n\u003cp\u003eYour %APP_NAME% team\u003c/p\u003e",
								"bodyFormat": "HTML",
								"replyTo": "noreply"
							},
							"changeEmailTemplate": {
								"senderLocalPart": "noreply",
								"subject": "Your sign-in email was changed for %APP_NAME%",
								"body": "\u003cp\u003eHello %DISPLAY_NAME%,\u003c/p\u003e\n\u003cp\u003eYour sign-in email for %APP_NAME% was changed to %NEW_EMAIL%.\u003c/p\u003e\n\u003cp\u003eIf you didn’t ask to change your email, follow this link to reset your sign-in email.\u003c/p\u003e\n\u003cp\u003e\u003ca href='%LINK%'\u003e%LINK%\u003c/a\u003e\u003c/p\u003e\n\u003cp\u003eThanks,\u003c/p\u003e\n\u003cp\u003eYour %APP_NAME% team\u003c/p\u003e",
								"bodyFormat": "HTML",
								"replyTo": "noreply"
							},
							"dnsInfo": {
								"customDomainState": "NOT_STARTED",
								"domainVerificationRequestTime": "1970-01-01T00:00:00Z"
							},
							"revertSecondFactorAdditionTemplate": {
								"senderLocalPart": "noreply",
								"subject": "You've added 2 step verification to your %APP_NAME% account.",
								"body": "\u003cp\u003eHello %DISPLAY_NAME%,\u003c/p\u003e\n\u003cp\u003eYour account in %APP_NAME% has been updated with a phone number %SECOND_FACTOR% for 2-step verification.\u003c/p\u003e\n\u003cp\u003eIf you didn't add this phone number for 2-step verification, click the link below to remove it.\u003c/p\u003e\n\u003cp\u003e\u003ca href='%LINK%'\u003e%LINK%\u003c/a\u003e\u003c/p\u003e\n\u003cp\u003eThanks,\u003c/p\u003e\n\u003cp\u003eYour %APP_NAME% team\u003c/p\u003e",
								"bodyFormat": "HTML",
								"replyTo": "noreply"
							}
						},
						"sendSms": {
							"smsTemplate": {
								"content": "%LOGIN_CODE% is your verification code for %APP_NAME%."
							}
						},
						"defaultLocale": "en"
					},
					"quota": {},
					"monitoring": {
						"requestLogging": {}
					},
					"multiTenant": {},
					"authorizedDomains": [
						"my-account.sidkik.app",
						"admin-my-account.sidkik.app",
						"yay.com"
					],
					"subtype": "FIREBASE_AUTH",
					"client": {
						"apiKey": "AIzaSyCYV9grthj5ZA_E8lZmMmoUF8-KpOEkg6Y",
						"permissions": {},
						"firebaseSubdomain": "acct-1tcgeonjjgqi"
					},
					"mfa": {
						"state": "DISABLED"
					},
					"blockingFunctions": {}
				}
				`,
			},
			want: true,
		},
		{
			name: "empty",
			args: args{
				response: `{
					"name": "projects/125051500756/config",
					"signIn": {
						"hashConfig": {
							"algorithm": "SCRYPT",
							"signerKey": "Rqol0YtxtvFoOn28eAg4/W7UkIfrBypR5mmsZy7e1JydqX8M3zm2oc+osHn9j4yh9fKO/27KhlJm25mgBg5Ebw==",
							"saltSeparator": "Bw==",
							"rounds": 8,
							"memoryCost": 14
						}
					},
					"notification": {
						"sendEmail": {
							"method": "DEFAULT",
							"resetPasswordTemplate": {
								"senderLocalPart": "noreply",
								"subject": "Reset your password for %APP_NAME%",
								"body": "\u003cp\u003eHello,\u003c/p\u003e\n\u003cp\u003eFollow this link to reset your %APP_NAME% password for your %EMAIL% account.\u003c/p\u003e\n\u003cp\u003e\u003ca href='%LINK%'\u003e%LINK%\u003c/a\u003e\u003c/p\u003e\n\u003cp\u003eIf you didn’t ask to reset your password, you can ignore this email.\u003c/p\u003e\n\u003cp\u003eThanks,\u003c/p\u003e\n\u003cp\u003eYour %APP_NAME% team\u003c/p\u003e",
								"bodyFormat": "HTML",
								"replyTo": "noreply"
							},
							"verifyEmailTemplate": {
								"senderLocalPart": "noreply",
								"subject": "Verify your email for %APP_NAME%",
								"body": "\u003cp\u003eHello %DISPLAY_NAME%,\u003c/p\u003e\n\u003cp\u003eFollow this link to verify your email address.\u003c/p\u003e\n\u003cp\u003e\u003ca href='%LINK%'\u003e%LINK%\u003c/a\u003e\u003c/p\u003e\n\u003cp\u003eIf you didn’t ask to verify this address, you can ignore this email.\u003c/p\u003e\n\u003cp\u003eThanks,\u003c/p\u003e\n\u003cp\u003eYour %APP_NAME% team\u003c/p\u003e",
								"bodyFormat": "HTML",
								"replyTo": "noreply"
							},
							"changeEmailTemplate": {
								"senderLocalPart": "noreply",
								"subject": "Your sign-in email was changed for %APP_NAME%",
								"body": "\u003cp\u003eHello %DISPLAY_NAME%,\u003c/p\u003e\n\u003cp\u003eYour sign-in email for %APP_NAME% was changed to %NEW_EMAIL%.\u003c/p\u003e\n\u003cp\u003eIf you didn’t ask to change your email, follow this link to reset your sign-in email.\u003c/p\u003e\n\u003cp\u003e\u003ca href='%LINK%'\u003e%LINK%\u003c/a\u003e\u003c/p\u003e\n\u003cp\u003eThanks,\u003c/p\u003e\n\u003cp\u003eYour %APP_NAME% team\u003c/p\u003e",
								"bodyFormat": "HTML",
								"replyTo": "noreply"
							},
							"dnsInfo": {
								"customDomainState": "NOT_STARTED",
								"domainVerificationRequestTime": "1970-01-01T00:00:00Z"
							},
							"revertSecondFactorAdditionTemplate": {
								"senderLocalPart": "noreply",
								"subject": "You've added 2 step verification to your %APP_NAME% account.",
								"body": "\u003cp\u003eHello %DISPLAY_NAME%,\u003c/p\u003e\n\u003cp\u003eYour account in %APP_NAME% has been updated with a phone number %SECOND_FACTOR% for 2-step verification.\u003c/p\u003e\n\u003cp\u003eIf you didn't add this phone number for 2-step verification, click the link below to remove it.\u003c/p\u003e\n\u003cp\u003e\u003ca href='%LINK%'\u003e%LINK%\u003c/a\u003e\u003c/p\u003e\n\u003cp\u003eThanks,\u003c/p\u003e\n\u003cp\u003eYour %APP_NAME% team\u003c/p\u003e",
								"bodyFormat": "HTML",
								"replyTo": "noreply"
							}
						},
						"sendSms": {
							"smsTemplate": {
								"content": "%LOGIN_CODE% is your verification code for %APP_NAME%."
							}
						},
						"defaultLocale": "en"
					},
					"quota": {},
					"monitoring": {
						"requestLogging": {}
					},
					"multiTenant": {},
					"authorizedDomains": [
						"my-account.sidkik.app",
						"admin-my-account.sidkik.app",
						"yay.com"
					],
					"subtype": "FIREBASE_AUTH",
					"client": {
						"apiKey": "AIzaSyCYV9grthj5ZA_E8lZmMmoUF8-KpOEkg6Y",
						"permissions": {},
						"firebaseSubdomain": "acct-1tcgeonjjgqi"
					},
					"mfa": {
						"state": "DISABLED"
					},
					"blockingFunctions": {}
				}
				`,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]interface{}
			// Unmarshal or Decode the JSON to the interface.
			json.Unmarshal([]byte(tt.args.response), &result)
			if got := flattenEmail(result["signIn"], nil, nil); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("flattenEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}
