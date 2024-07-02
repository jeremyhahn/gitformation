# gitformation

Binds git commit changes with AWS CloudFormation stack operations, by parseing a git log to determine which files
have been created, modified and/or deleted since the last commit. For each change, the corresponding AWS CloudFormation
stack operation is executed. For new files, create-stack, updated files, update-stack, and deleted files, delete-stack.

# Build

### github

Use the following git configuration option to enable private github repository access via SSH:

    git config --global url."git@github.com:".insteadOf "https://github.com/"

# Examples

    # Use custom AWS Profile for deployment
    gitformation manage-stacks --debug --profile=my-custom-profile

    # Use dynamic AWS Profile for deployment (mycompany-preprod)
    gitformation manage-stacks --debug --env preprod --profile-prefix=mycompany

    # Use pattern matcher to process all files in the repository
    gitformation manage-stacks --debug  -filter=[a-zA-Z0-9./]+

    # Use pattern matcher to process changes only in the examples folder
    gitformation manage-stacks --debug --filter=examples/*


## Support

Please consider supporting this project for ongoing success and sustainability. I'm a passionate open source contributor making a professional living creating free, secure, scalable, robust, enterprise grade, distributed systems and cloud native solutions.

I'm also available for international consulting opportunities. Please let me know how I can assist you or your organization in achieving your desired security posture and technology goals.

https://github.com/sponsors/jeremyhahn

https://www.linkedin.com/in/jeremyhahn