#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <set>
#include <cmath>
#include <sstream>
#include <algorithm>
#include <numeric>

using namespace std;

// Not: "what does the user want"
// But: "what does this utterance reveal about who is speaking"

static const map<string, vector<string>> CLASS_VOCAB = {
    {"reveals_fear", {
        "scared","afraid","worry","anxious","fear","nervous","terrified",
        "dread","panic","safe","danger","hurt","pain","please","help",
        "sorry","uncertain","lost","alone","never","nobody","protect",
        "wrong","bad","end","disappear","gone","always","everyone"
    }},
    {"reveals_certainty", {
        "know","certain","sure","obvious","clearly","definitely","must",
        "should","fact","truth","real","correct","right","exactly",
        "absolutely","simple","just","only","nothing","everything",
        "everyone","because","obviously","always","never","clearly"
    }},
    {"reveals_search", {
        "what","why","how","where","when","who","which","looking",
        "searching","find","understand","mean","explain","trying",
        "wonder","curious","question","think","maybe","perhaps",
        "could","might","possible","sense","confused","unsure","seeking"
    }},
    {"reveals_refusal", {
        "no","not","refuse","stop","enough","done","leave","away",
        "pointless","useless","fine","whatever","forget","ignore",
        "quiet","bother","going","doing","saying","nothing",
        "won't","don't","can't","wont","dont","cant","nope"
    }},
    {"reveals_recognition", {
        "recognize","realize","notice","familiar","remind","same",
        "understand","see","know","that's","been","felt","makes",
        "sense","yourself","like","this","you","your","always",
        "you do","you are","you always","i see you","i know you"
    }},
    {"reveals_mirror", {
        "tell","answer","your","turn","first","back","asking",
        "deflect","avoiding","answering","same","you","doing",
        "instead","around","said","why","you tell","you answer",
        "your turn","you first","back to you","same to you",
        "you're deflecting","you're avoiding","you never answer"
    }}
};

vector<string> tokenize(const string& text) {
    vector<string> tokens;
    string lower = text;
    transform(lower.begin(), lower.end(), lower.begin(), ::tolower);
    istringstream iss(lower);
    string token;
    while (iss >> token) {
        // Strip punctuation
        string clean;
        for (char c : token)
            if (!ispunct((unsigned char)c)) clean += c;
        if (!clean.empty()) tokens.push_back(clean);
    }
    return tokens;
}

// Compute document frequency across all classes
map<string, int> build_doc_freq() {
    map<string, int> df;
    for (const auto& [cls, vocab] : CLASS_VOCAB) {
        set<string> seen(vocab.begin(), vocab.end());
        for (const auto& term : seen) df[term]++;
    }
    return df;
}

int main() {
    string text;
    getline(cin, text);

    if (text.empty()) {
        cout << "reveals_nothing" << endl;
        return 0;
    }

    vector<string> tokens = tokenize(text);
    if (tokens.empty()) {
        cout << "reveals_nothing" << endl;
        return 0;
    }

    map<string, int> input_count;
    for (const auto& t : tokens) input_count[t]++;

    static const auto doc_freq = build_doc_freq();
    const int N = (int)CLASS_VOCAB.size();

    map<string, double> scores;
    for (const auto& [cls, vocab] : CLASS_VOCAB) {
        double score = 0.0;
        for (const auto& term : vocab) {
            auto it = input_count.find(term);
            if (it != input_count.end()) {
                double tf = (double)it->second / tokens.size();
                double df_val = doc_freq.count(term) ? doc_freq.at(term) : 1;
                double idf = log((double)(N + 1) / (df_val + 1)) + 1.0;
                score += tf * idf;
            }
        }
        scores[cls] = score;
    }

    auto best = max_element(scores.begin(), scores.end(),
        [](const auto& a, const auto& b){ return a.second < b.second; });

    if (best->second < 0.001) {
        cout << "reveals_nothing" << endl;
    } else {
        cout << best->first << endl;
    }

    return 0;
}
