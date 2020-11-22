#Prom - find a prominent image on the website

Simple go service to find a prominent image on the website. 
Script is also capable of extracting excerpt from the website, and content with HTML tags.
It's useful to create a website "preview" for news reader or something.

##Usage

    go run main.go
    
then try in browser

    https://localhost:9090/url/?url=https://github.com/slav123/
    
    
sample output:
	
    {
        "title":"slav123 (Slawomir Jasinski) · GitHub",
        "date_published":"",
        "lead_image_url":"https://avatars1.githubusercontent.com/u/185637?s=400&u=d4ba7571ac4c302ccb67f9962727d1c6fa01e170&v=4",
        "dek":"We use optional third-party analytics cookies to ...",
        "url":"https://github.com/slav123/",
        "domain":"https://github.com",
        "excerpt":"We use optional third-party analytics cookies to understand how you use GitHub.com so we can build better products",
        "content":"<div><div> <div> <p> We use optional third-party analytics cookies to understand how you use GitHub.com so we can... </p> </div> </div></div>"
    }

  
## @2do

* read schema.org info
* recognize meta from wordpress
* cleanup content
* try to skip cookies