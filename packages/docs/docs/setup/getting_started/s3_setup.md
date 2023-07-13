# Amazon S3 Setup Guide
S3 (Simple Storage Service) is a cloud storage service provided by Amazon Web Services (AWS). It is a simple and secure way to store and retrieve any amount of data from anywhere on the web. This guide will show you how to set up an S3 bucket for use in this application. 

Currently, we use this for storing profile and banner images for users and organizations. This means that we use a public read bucket policy, which allows anyone to read the contents of the bucket. If you want to use S3 for storing private data, you will need to add an additional bucket with a private read policy.

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


*Refer to the official Amazon S3 and AWS SDK documentation for more detailed information and assistance.*