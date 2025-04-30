# PowerShell script: generate-cert.ps1

Write-Host "Creating server.key (RSA 2048)"
openssl genrsa -out server.key 2048

# Dacă dorești în schimb cheia EC (nu ambele!), comentează linia de mai sus și decomentează cea de jos:
# Write-Host "Creating server.key (EC secp384r1)"
# openssl ecparam -genkey -name secp384r1 -out server.key

Write-Host "Creating server.crt (valid 365 days)"
openssl req -new -x509 -sha256 -key server.key -out server.crt -batch -days 365
