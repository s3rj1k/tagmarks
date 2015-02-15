#!/usr/bin/perl

use DBI;
use strict;
use warnings;

my $Url;
my $Name;
my $Tag;
my $mDate;
my $aDate;
my $Date;
my @Tag;
my $iTag;
my $currLine;

my $dbh = DBI->connect("DBI:SQLite:dbname=test.db", "", "", { RaiseError => 1, AutoCommit => 0 }) or die $DBI::errstr;
$dbh->do("PRAGMA synchronous=OFF");
$dbh->do("DROP TABLE IF EXISTS tagmarks");
$dbh->do("CREATE VIRTUAL TABLE tagmarks USING fts4 (url TEXT NOT NULL, name TEXT NOT NULL, date TEXT NOT NULL, tags TEXT DEFAULT ('NULL'),);") or die $DBI::errstr;
my $sth = $dbh->prepare('INSERT INTO tagmarks (url,name,date,tags) VALUES (?, ?, ?, ?)');

while (<STDIN>) {
	$currLine = $_;
	chomp $currLine;
	if($currLine =~ m/(<A HREF="[^"]*")/) {
		$Url = $1;
		$Url =~ s/<A HREF=//gi;
		$Url =~ s/"//gi;
		if($currLine =~ m/(>[^"]*<\/A>)/) {
			$Name = $1;
			$Name =~ s/<\/A>//gi;
			$Name =~ s/>//gi;
		} else {
			$Name = "NULL";
		}
		if($currLine =~ m/(TAGS="[^"]*")/) {
			$Tag = $1;
			$Tag =~ s/TAGS=//gi;
			$Tag =~ s/"//gi;
		} else {
			$Tag = "NULL";
		}
		if($currLine =~ m/(LAST_MODIFIED="[^"]*")/) {
			$mDate = $1;
			$mDate =~ s/LAST_MODIFIED=//gi;
			$mDate =~ s/"//gi;
		} else {
			$mDate = time;
		}
		if($currLine =~ m/(ADD_DATE="[^"]*")/) {
			$aDate = $1;
			$aDate =~ s/ADD_DATE=//gi;
			$aDate =~ s/"//gi;
		} else {
			$aDate = time;
		}
		if ($mDate <= $aDate) {
			$Date = $aDate;
		} elsif ($mDate > $aDate) {
			$Date = $mDate;
		}
		@Tag = split(/,/, $Tag);
		@Tag = sort @Tag;
		$Tag = join(",", @Tag);
		$sth->execute ($Url, $Name, $Date, $Tag) or die $DBI::errstr;
		print "$Url, $Name, $Date, $Tag\n";
	}
}
$dbh->commit();
$dbh->disconnect();
