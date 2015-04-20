# zork-ai
A wrapper around Zork I that should be able to learn from a human player and assist.

### Setup

* Compile [Frotz](https://github.com/DavidGriffith/frotz/) in dumb mode.
```sh
git clone https://github.com/DavidGriffith/frotz
cd frotz
make dumb
make install_dumb # optional
```
* Download Zork I from [Infocom's website](http://www.infocom-if.org/downloads/downloads.html) and extract `DATA/ZORK1.DAT`.

### How to Run
```sh
go run zork.go <path to dfrotz> <path to ZORK1.DAT>
```
