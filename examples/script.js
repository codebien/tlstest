import httptls from "k6/x/httptls";

export const options = {
  iterations: 1,
};

export default async function () {
  const v = await httptls.isExpired("github.com");
  console.log(`expired: ${v}`);
  // expired: false

  const certs = await httptls.chain("github.com");
  console.log(JSON.stringify(certs, "", "  "));

  // [
  //   {
  //     "subject": "CN=github.com",
  //     "expires": 1770335999000,
  //     "isca": false
  //   },
  //   {
  //     "subject": "CN=Sectigo ECC Domain Validation Secure Server CA,O=Sectigo Limited,L=Salford,ST=Greater Manchester,C=GB",
  //     "expires": 1924991999000,
  //     "isca": true
  //   },
  //   {
  //     "subject": "CN=USERTrust ECC Certification Authority,O=The USERTRUST Network,L=Jersey City,ST=New Jersey,C=US",
  //     "expires": 1861919999000,
  //     "isca": true
  //   }
  // ]
}
