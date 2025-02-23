## Basic

A trending algo over simplified based on reactions

## Score

Add a trending score calculation that considers time decay and multiple engagement types.

Key improvements:

Considers multiple engagement types with different weights
Implements time decay using a power law function
Adds a base time offset (7200 seconds = 2 hours) to smooth out early scoring
More comprehensive trending score instead of just raw counts
The time decay formula is similar to HackerNews' algorithm but adjusted for shorter-term trending content. You might want to tune the weights and decay factors based on your specific needs.