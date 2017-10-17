Features

Implement a user manager system for user to login and edit their profiles. Users can login on web page. 

After a user inputs his/her username and password, the backend system will authenticate his/her identity. If the login is successful, the relevant user information is displayed, otherwise an error message will be shown.

After a successful login, a user can edit the following information: 


Upload a picture as his/her profile picture 
Change his/her nickname (support unicode characters with utf­-8 encoding) 


User information includes: username (cannot be changed), nickname, profile picture.  For test, the initial user data can be directly insert into database. Make sure there are at least 10 million user accounts in the test database.

Performance


Supports up to 1000 login requests per second (from at least 200 unique users)
Supports up to 1000 concurrent http requests


Design Constraint


Separate HTTP server and TCP server and put the main logic on TCP server
Backend authentication logic should be done in the TCP server
User information must be stored in a MySQL database. Connect by MySQL Go client.
Use standard library whenever possible.
Web interface will not directly connect to MySQL. For each HTTP request, web interface will send a TCP request to the TCP server, which will handle business logic and query the database.
