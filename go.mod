module github.com/bsdlp/what-im-watching

go 1.15

require (
	github.com/99designs/gqlgen v0.13.0
	github.com/Yamashou/gqlgenc v0.0.0-20201118110422-c5e4e29019a7
	github.com/aws/aws-lambda-go v1.20.0
	github.com/dghubble/go-twitter v0.0.0-20201011215211-4b180d0cc78d
	github.com/dghubble/oauth1 v0.6.0
	github.com/kelseyhightower/envconfig v1.4.0
	golang.org/x/tools v0.0.0-20201118215654-4d9c4f8a78b0 // indirect
)

replace github.com/kelseyhightower/envconfig v1.4.0 => github.com/bsdlp/envconfig v1.5.0
