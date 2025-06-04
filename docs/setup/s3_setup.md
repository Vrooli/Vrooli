# Amazon S3 Setup Guide
S3 (Simple Storage Service) is a cloud storage service provided by Amazon Web Services (AWS). It is a simple and secure way to store and retrieve any amount of data from anywhere on the web. This guide will show you how to set up an S3 bucket for use in this application. 

Currently, we use this for storing profile and banner images for users and teams. This means that we use a public read bucket policy, which allows anyone to read the contents of the bucket. If you want to use S3 for storing private data, you will need to add an additional bucket with a private read policy.

## Step 1: Create an AWS account
1. Visit the AWS homepage at [https://aws.amazon.com/](https://aws.amazon.com/).
2. Click on the "Create an AWS Account" button at the top right of the page.
3. Follow the on-screen instructions to complete the account setup process.

## Step 2: Create an S3 Bucket
1. Once you've signed into the AWS Management Console, navigate to the Amazon S3 service. You can find this under the "Storage" category.
2. Click on "Create bucket".
3. Give your bucket a unique name. The application should be updated to reflect this. Search for `BUCKET_NAME` in the project to find the relevant code.
4. Choose the region where you want your bucket to reside. The application should be updated to reflect this. Search for `S3Client(` or `REGION` in the project to find the relevant code.
5. Choose "ACLs disabled" for the "Object Ownership" setting.
6. Configure options and set permissions as needed, then click "Create".
7. Uncheck boxes that block public access. This will allow anyone to read the contents of the bucket.
8. Set "Bucket Versioning" to "Disabled", since it is not needed for our purposes.
9. Set the encryption type to "SSE-S3", and the bucket key to "Enable".

## Step 3: Create an IAM User and Obtain AWS Credentials
1. Navigate to the IAM (Identity and Access Management) service from the AWS Management Console.
2. Click on "Users" in the left sidebar and then "Add User".
3. Enter a username and click "Next".
4. Select "Add user to group" and click "Create group".
5. Enter a group name and click "Create user group".
6. Now you should be back on the "Add user" page. Select the group you just created and click "Next".
7. Review the user and group, then click "Create user".
8. Click on user, and go to the "Security credentials" tab.
9. Click "Create access key".
10. Select the "Application running outside AWS" option and click "Next".
11. Click "Download .csv file" to download your credentials. Make sure to securely store these credentials, as you won't be able to access the secret access key again.

## Step 4: Store Credentials in .env and .env-prod
1. Open `.env` and `.env-prod`. Replace `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` with the credentials you downloaded earlier.
2. Save and close the files.

## Step 5: Set Bucket Policy
1. Navigate to the bucket you created earlier.
2. Click on the "Permissions" tab.
3. Click on "Edit" in the "Bucket Policy" section.
4. You can use the [AWS Policy Generator](https://awspolicygen.s3.amazonaws.com/policygen.html) or manually craft the JSON policy. The policy should grant your IAM user permissions to manage objects within the bucket and to list the bucket itself. It should also grant public read access to the objects if desired.
5. Select "S3 Bucket Policy" for the "Select Type of Policy" option if using the generator.
6. Select "Allow" for the "Effect" option.
7. For "Principal", you'll use the ARN of the IAM user created in Step 3.
    a. Navigate to the IAM service from the AWS Management Console.
    b. Click on "Users" in the sidebar and then click on the user you created (e.g., `vrooli-user`).
    c. Copy the "User ARN" at the top of the page (e.g., `arn:aws:iam::123456789012:user/your-user-name`).
8. For "Actions", your IAM user will need:
    *   `s3:DeleteObject`, `s3:GetObject`, `s3:PutObject`, `s3:PutObjectAcl` for object operations.
    *   `s3:ListBucket` for bucket operations (like the health check).
9. For "Amazon Resource Name (ARN)":
    *   For object actions, use your bucket's ARN followed by `/*` (e.g., `arn:aws:s3:::vrooli-bucket/*`).
    *   For the `s3:ListBucket` action, use your bucket's ARN without the `/*` (e.g., `arn:aws:s3:::vrooli-bucket`).
10. Click "Add Statement" for each part of the policy if using the generator, then "Generate Policy". The combined policy should look something like this, replacing placeholders with your actual ARNs and bucket name:

```json
{
    "Version": "2012-10-17",
    "Id": "VrooliBucketPolicy",
    "Statement": [
        {
            "Sid": "AllowUserObjectActions",
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::YOUR_AWS_ACCOUNT_ID:user/YOUR_IAM_USER_NAME"
            },
            "Action": [
                "s3:DeleteObject",
                "s3:GetObject",
                "s3:PutObject",
                "s3:PutObjectAcl"
            ],
            "Resource": "arn:aws:s3:::YOUR_BUCKET_NAME/*"
        },
        {
            "Sid": "AllowUserListBucket",
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::YOUR_AWS_ACCOUNT_ID:user/YOUR_IAM_USER_NAME"
            },
            "Action": "s3:ListBucket",
            "Resource": "arn:aws:s3:::YOUR_BUCKET_NAME"
        },
        {
            "Sid": "AddPublicReadAccess",
            "Effect": "Allow",
            "Principal": "*",
            "Action": "s3:GetObject",
            "Resource": "arn:aws:s3:::YOUR_BUCKET_NAME/*"
        }
    ]
}
```
11. Paste the complete policy into the "Bucket policy" editor on the S3 console and click "Save changes".

*Refer to the official Amazon S3 and AWS SDK documentation for more detailed information and assistance.*