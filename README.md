# Prom - find a prominent image on the website

Simple go service to find a prominent image on the website. 

App is also capable of extracting excerpt from the website, and content with HTML tags.
It's useful to create a website "preview" for news reader or something.
## How it works

Algorithm is pretty simple, we scrape all images. We look for the biggest one. It utilises go routines to do the image comparison.

We do some smart image type recognition, and we don't download whole images, only headers to check image sizes. 

## Usage

    go run main.go
    
then try in browser

    http://localhost:9999/url/?url=https://www.jasinski.us
    
    
sample output:
	```JSON
{
    "title": "about |\nSlawomir Jasinski",
    "success": true,
    "message": "Content extracted successfully",
    "date_published": "",
    "last_modified": "Fri, 20 Dec 2024 09:48:48 GMT",
    "lead_image_url": "https://www.jasinski.us/images/2024/06/aws-advanced-services-for-solutions-architects.jpeg",
    "dek": "Shifting to a cloud-native architecture 🌐 is not just a trend—it’s a strategic move that can propel your applications to new heights in terms of flexibility, scalability, and performance. Today, we’re diving deep into the when, why, and how of making this crucial transition, with an eye on navigating the choices between cloud-native and traditional setups 🚀. Let’s also tackle a common concern: the risk of vendor lock-in, and explore “safe” strategies to mitigate this issue.",
    "url": "https://www.jasinski.us",
    "domain": "https://www.jasinski.us",
    "excerpt": "Shifting to a cloud-native architecture 🌐 is not just a trend—it’s a strategic move that can propel your applications to new heights in terms of flexibility, scalability, and performance. Today, we’re diving deep into the when, why, and how of making this crucial transition, with an eye on navigating the choices between cloud-native and traditional setups 🚀. Let’s also tackle a common concern: the risk of vendor lock-in, and explore",
    "content": "<p>Shifting to a cloud-native architecture 🌐 is not just a trend—it’s a strategic move that can propel your applications to new heights in terms of flexibility, scalability, and performance. Today, we’re diving deep into the when, why, and how of making this crucial transition, with an eye on navigating the choices between cloud-native and traditional setups 🚀. Let’s also tackle a common concern: the risk of vendor lock-in, and explore “safe” strategies to mitigate this issue.</p>"
}
```

  
## Docker

The application is available as a Docker container on Docker Hub at `slav123/prom`. You can pull and run it using:

```bash
docker pull slav123/prom:latest
docker run -p 9090:9090 slav123/prom
```

## Recent Changes (2025-01-02)

* Fixed image type detection in `imageutils.DetermineImageType` function
* Added proper handling of SVG detection using the `bytes` package
* Added GitHub Actions workflow for automatic Docker image deployment
* Images are now automatically built and pushed to Docker Hub on every push to master branch

## @2do

* read schema.org info
* recognize meta from wordpress
* cleanup content
* try to skip cookies warning
* webp