export const projectOrTeam_findMany = {
  "fieldName": "projectOrTeams",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectOrTeams",
        "loc": {
          "start": 2024,
          "end": 2038
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2039,
              "end": 2044
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 2047,
                "end": 2052
              }
            },
            "loc": {
              "start": 2046,
              "end": 2052
            }
          },
          "loc": {
            "start": 2039,
            "end": 2052
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "edges",
              "loc": {
                "start": 2060,
                "end": 2065
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "cursor",
                    "loc": {
                      "start": 2076,
                      "end": 2082
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2076,
                    "end": 2082
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 2091,
                      "end": 2095
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Project",
                            "loc": {
                              "start": 2117,
                              "end": 2124
                            }
                          },
                          "loc": {
                            "start": 2117,
                            "end": 2124
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Project_list",
                                "loc": {
                                  "start": 2146,
                                  "end": 2158
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2143,
                                "end": 2158
                              }
                            }
                          ],
                          "loc": {
                            "start": 2125,
                            "end": 2172
                          }
                        },
                        "loc": {
                          "start": 2110,
                          "end": 2172
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Team",
                            "loc": {
                              "start": 2192,
                              "end": 2196
                            }
                          },
                          "loc": {
                            "start": 2192,
                            "end": 2196
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Team_list",
                                "loc": {
                                  "start": 2218,
                                  "end": 2227
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2215,
                                "end": 2227
                              }
                            }
                          ],
                          "loc": {
                            "start": 2197,
                            "end": 2241
                          }
                        },
                        "loc": {
                          "start": 2185,
                          "end": 2241
                        }
                      }
                    ],
                    "loc": {
                      "start": 2096,
                      "end": 2251
                    }
                  },
                  "loc": {
                    "start": 2091,
                    "end": 2251
                  }
                }
              ],
              "loc": {
                "start": 2066,
                "end": 2257
              }
            },
            "loc": {
              "start": 2060,
              "end": 2257
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 2262,
                "end": 2270
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 2281,
                      "end": 2292
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2281,
                    "end": 2292
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 2301,
                      "end": 2317
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2301,
                    "end": 2317
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorTeam",
                    "loc": {
                      "start": 2326,
                      "end": 2339
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2326,
                    "end": 2339
                  }
                }
              ],
              "loc": {
                "start": 2271,
                "end": 2345
              }
            },
            "loc": {
              "start": 2262,
              "end": 2345
            }
          }
        ],
        "loc": {
          "start": 2054,
          "end": 2349
        }
      },
      "loc": {
        "start": 2024,
        "end": 2349
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 32,
          "end": 34
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 32,
        "end": 34
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 35,
          "end": 45
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 35,
        "end": 45
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 46,
          "end": 56
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 46,
        "end": 56
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 57,
          "end": 62
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 57,
        "end": 62
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 63,
          "end": 68
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 63,
        "end": 68
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 69,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 88,
                  "end": 92
                }
              },
              "loc": {
                "start": 88,
                "end": 92
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 106,
                      "end": 114
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 103,
                    "end": 114
                  }
                }
              ],
              "loc": {
                "start": 93,
                "end": 120
              }
            },
            "loc": {
              "start": 81,
              "end": 120
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 132,
                  "end": 136
                }
              },
              "loc": {
                "start": 132,
                "end": 136
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 150,
                      "end": 158
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 147,
                    "end": 158
                  }
                }
              ],
              "loc": {
                "start": 137,
                "end": 164
              }
            },
            "loc": {
              "start": 125,
              "end": 164
            }
          }
        ],
        "loc": {
          "start": 75,
          "end": 166
        }
      },
      "loc": {
        "start": 69,
        "end": 166
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 167,
          "end": 170
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 177,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 177,
              "end": 186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 191,
                "end": 200
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 191,
              "end": 200
            }
          }
        ],
        "loc": {
          "start": 171,
          "end": 202
        }
      },
      "loc": {
        "start": 167,
        "end": 202
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 240,
          "end": 248
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 255,
                "end": 267
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 278,
                      "end": 280
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 278,
                    "end": 280
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 289,
                      "end": 297
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 289,
                    "end": 297
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 306,
                      "end": 317
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 306,
                    "end": 317
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 326,
                      "end": 330
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 326,
                    "end": 330
                  }
                }
              ],
              "loc": {
                "start": 268,
                "end": 336
              }
            },
            "loc": {
              "start": 255,
              "end": 336
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 341,
                "end": 343
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 341,
              "end": 343
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 348,
                "end": 358
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 348,
              "end": 358
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 363,
                "end": 373
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 363,
              "end": 373
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 378,
                "end": 394
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 378,
              "end": 394
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 399,
                "end": 407
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 399,
              "end": 407
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 412,
                "end": 421
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 412,
              "end": 421
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 426,
                "end": 438
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 426,
              "end": 438
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 443,
                "end": 459
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 443,
              "end": 459
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 464,
                "end": 474
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 464,
              "end": 474
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 479,
                "end": 491
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 479,
              "end": 491
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 496,
                "end": 508
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 496,
              "end": 508
            }
          }
        ],
        "loc": {
          "start": 249,
          "end": 510
        }
      },
      "loc": {
        "start": 240,
        "end": 510
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 511,
          "end": 513
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 511,
        "end": 513
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 514,
          "end": 524
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 514,
        "end": 524
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 525,
          "end": 535
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 525,
        "end": 535
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 536,
          "end": 545
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 536,
        "end": 545
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 546,
          "end": 557
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 546,
        "end": 557
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 558,
          "end": 564
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 574,
                "end": 584
              }
            },
            "directives": [],
            "loc": {
              "start": 571,
              "end": 584
            }
          }
        ],
        "loc": {
          "start": 565,
          "end": 586
        }
      },
      "loc": {
        "start": 558,
        "end": 586
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 587,
          "end": 592
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 606,
                  "end": 610
                }
              },
              "loc": {
                "start": 606,
                "end": 610
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 624,
                      "end": 632
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 621,
                    "end": 632
                  }
                }
              ],
              "loc": {
                "start": 611,
                "end": 638
              }
            },
            "loc": {
              "start": 599,
              "end": 638
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 650,
                  "end": 654
                }
              },
              "loc": {
                "start": 650,
                "end": 654
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 668,
                      "end": 676
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 665,
                    "end": 676
                  }
                }
              ],
              "loc": {
                "start": 655,
                "end": 682
              }
            },
            "loc": {
              "start": 643,
              "end": 682
            }
          }
        ],
        "loc": {
          "start": 593,
          "end": 684
        }
      },
      "loc": {
        "start": 587,
        "end": 684
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 685,
          "end": 696
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 685,
        "end": 696
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 697,
          "end": 711
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 697,
        "end": 711
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 712,
          "end": 717
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 712,
        "end": 717
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 718,
          "end": 727
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 718,
        "end": 727
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 728,
          "end": 732
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 742,
                "end": 750
              }
            },
            "directives": [],
            "loc": {
              "start": 739,
              "end": 750
            }
          }
        ],
        "loc": {
          "start": 733,
          "end": 752
        }
      },
      "loc": {
        "start": 728,
        "end": 752
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 753,
          "end": 767
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 753,
        "end": 767
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 768,
          "end": 773
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 768,
        "end": 773
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 774,
          "end": 777
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 784,
                "end": 793
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 784,
              "end": 793
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 798,
                "end": 809
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 798,
              "end": 809
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 814,
                "end": 825
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 814,
              "end": 825
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 830,
                "end": 839
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 830,
              "end": 839
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 844,
                "end": 851
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 844,
              "end": 851
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 856,
                "end": 864
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 856,
              "end": 864
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 869,
                "end": 881
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 869,
              "end": 881
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 886,
                "end": 894
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 886,
              "end": 894
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 899,
                "end": 907
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 899,
              "end": 907
            }
          }
        ],
        "loc": {
          "start": 778,
          "end": 909
        }
      },
      "loc": {
        "start": 774,
        "end": 909
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 939,
          "end": 941
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 939,
        "end": 941
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 942,
          "end": 952
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 942,
        "end": 952
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 953,
          "end": 956
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 953,
        "end": 956
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 957,
          "end": 966
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 957,
        "end": 966
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 967,
          "end": 979
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 986,
                "end": 988
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 986,
              "end": 988
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 993,
                "end": 1001
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 993,
              "end": 1001
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1006,
                "end": 1017
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1006,
              "end": 1017
            }
          }
        ],
        "loc": {
          "start": 980,
          "end": 1019
        }
      },
      "loc": {
        "start": 967,
        "end": 1019
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1020,
          "end": 1023
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOwn",
              "loc": {
                "start": 1030,
                "end": 1035
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1030,
              "end": 1035
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1040,
                "end": 1052
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1040,
              "end": 1052
            }
          }
        ],
        "loc": {
          "start": 1024,
          "end": 1054
        }
      },
      "loc": {
        "start": 1020,
        "end": 1054
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1086,
          "end": 1088
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1086,
        "end": 1088
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 1089,
          "end": 1100
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1089,
        "end": 1100
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1101,
          "end": 1107
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1101,
        "end": 1107
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1108,
          "end": 1118
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1108,
        "end": 1118
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1119,
          "end": 1129
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1119,
        "end": 1129
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isOpenToNewMembers",
        "loc": {
          "start": 1130,
          "end": 1148
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1130,
        "end": 1148
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1149,
          "end": 1158
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1149,
        "end": 1158
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 1159,
          "end": 1172
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1159,
        "end": 1172
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "membersCount",
        "loc": {
          "start": 1173,
          "end": 1185
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1173,
        "end": 1185
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 1186,
          "end": 1198
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1186,
        "end": 1198
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 1199,
          "end": 1211
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1199,
        "end": 1211
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1212,
          "end": 1221
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1212,
        "end": 1221
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1222,
          "end": 1226
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 1236,
                "end": 1244
              }
            },
            "directives": [],
            "loc": {
              "start": 1233,
              "end": 1244
            }
          }
        ],
        "loc": {
          "start": 1227,
          "end": 1246
        }
      },
      "loc": {
        "start": 1222,
        "end": 1246
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 1247,
          "end": 1259
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1266,
                "end": 1268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1266,
              "end": 1268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 1273,
                "end": 1281
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1273,
              "end": 1281
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 1286,
                "end": 1289
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1286,
              "end": 1289
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1294,
                "end": 1298
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1294,
              "end": 1298
            }
          }
        ],
        "loc": {
          "start": 1260,
          "end": 1300
        }
      },
      "loc": {
        "start": 1247,
        "end": 1300
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1301,
          "end": 1304
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 1311,
                "end": 1324
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1311,
              "end": 1324
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1329,
                "end": 1338
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1329,
              "end": 1338
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1343,
                "end": 1354
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1343,
              "end": 1354
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 1359,
                "end": 1368
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1359,
              "end": 1368
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1373,
                "end": 1382
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1373,
              "end": 1382
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1387,
                "end": 1394
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1387,
              "end": 1394
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1399,
                "end": 1411
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1399,
              "end": 1411
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1416,
                "end": 1424
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1416,
              "end": 1424
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 1429,
                "end": 1443
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1454,
                      "end": 1456
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1454,
                    "end": 1456
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1465,
                      "end": 1475
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1465,
                    "end": 1475
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1484,
                      "end": 1494
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1484,
                    "end": 1494
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 1503,
                      "end": 1510
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1503,
                    "end": 1510
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 1519,
                      "end": 1530
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1519,
                    "end": 1530
                  }
                }
              ],
              "loc": {
                "start": 1444,
                "end": 1536
              }
            },
            "loc": {
              "start": 1429,
              "end": 1536
            }
          }
        ],
        "loc": {
          "start": 1305,
          "end": 1538
        }
      },
      "loc": {
        "start": 1301,
        "end": 1538
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1569,
          "end": 1571
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1569,
        "end": 1571
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 1572,
          "end": 1583
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1572,
        "end": 1583
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1584,
          "end": 1590
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1584,
        "end": 1590
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 1591,
          "end": 1603
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1591,
        "end": 1603
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1604,
          "end": 1607
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 1614,
                "end": 1627
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1614,
              "end": 1627
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1632,
                "end": 1641
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1632,
              "end": 1641
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1646,
                "end": 1657
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1646,
              "end": 1657
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 1662,
                "end": 1671
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1662,
              "end": 1671
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1676,
                "end": 1685
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1676,
              "end": 1685
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1690,
                "end": 1697
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1690,
              "end": 1697
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1702,
                "end": 1714
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1702,
              "end": 1714
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1719,
                "end": 1727
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1719,
              "end": 1727
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 1732,
                "end": 1746
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1757,
                      "end": 1759
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1757,
                    "end": 1759
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1768,
                      "end": 1778
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1768,
                    "end": 1778
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1787,
                      "end": 1797
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1787,
                    "end": 1797
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 1806,
                      "end": 1813
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1806,
                    "end": 1813
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 1822,
                      "end": 1833
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1822,
                    "end": 1833
                  }
                }
              ],
              "loc": {
                "start": 1747,
                "end": 1839
              }
            },
            "loc": {
              "start": 1732,
              "end": 1839
            }
          }
        ],
        "loc": {
          "start": 1608,
          "end": 1841
        }
      },
      "loc": {
        "start": 1604,
        "end": 1841
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1872,
          "end": 1874
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1872,
        "end": 1874
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1875,
          "end": 1885
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1875,
        "end": 1885
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1886,
          "end": 1896
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1886,
        "end": 1896
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 1897,
          "end": 1908
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1897,
        "end": 1908
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1909,
          "end": 1915
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1909,
        "end": 1915
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 1916,
          "end": 1921
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1916,
        "end": 1921
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 1922,
          "end": 1942
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1922,
        "end": 1942
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 1943,
          "end": 1947
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1943,
        "end": 1947
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 1948,
          "end": 1960
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1948,
        "end": 1960
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 10,
          "end": 20
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 24,
            "end": 29
          }
        },
        "loc": {
          "start": 24,
          "end": 29
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 32,
                "end": 34
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 32,
              "end": 34
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 35,
                "end": 45
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 35,
              "end": 45
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 46,
                "end": 56
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 46,
              "end": 56
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 57,
                "end": 62
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 57,
              "end": 62
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 63,
                "end": 68
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 63,
              "end": 68
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 69,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 88,
                        "end": 92
                      }
                    },
                    "loc": {
                      "start": 88,
                      "end": 92
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 106,
                            "end": 114
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 103,
                          "end": 114
                        }
                      }
                    ],
                    "loc": {
                      "start": 93,
                      "end": 120
                    }
                  },
                  "loc": {
                    "start": 81,
                    "end": 120
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 132,
                        "end": 136
                      }
                    },
                    "loc": {
                      "start": 132,
                      "end": 136
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 150,
                            "end": 158
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 147,
                          "end": 158
                        }
                      }
                    ],
                    "loc": {
                      "start": 137,
                      "end": 164
                    }
                  },
                  "loc": {
                    "start": 125,
                    "end": 164
                  }
                }
              ],
              "loc": {
                "start": 75,
                "end": 166
              }
            },
            "loc": {
              "start": 69,
              "end": 166
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 167,
                "end": 170
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 177,
                      "end": 186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 177,
                    "end": 186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 191,
                      "end": 200
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 191,
                    "end": 200
                  }
                }
              ],
              "loc": {
                "start": 171,
                "end": 202
              }
            },
            "loc": {
              "start": 167,
              "end": 202
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 204
        }
      },
      "loc": {
        "start": 1,
        "end": 204
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 214,
          "end": 226
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 230,
            "end": 237
          }
        },
        "loc": {
          "start": 230,
          "end": 237
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 240,
                "end": 248
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 255,
                      "end": 267
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 278,
                            "end": 280
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 278,
                          "end": 280
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 289,
                            "end": 297
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 289,
                          "end": 297
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 306,
                            "end": 317
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 306,
                          "end": 317
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 326,
                            "end": 330
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 326,
                          "end": 330
                        }
                      }
                    ],
                    "loc": {
                      "start": 268,
                      "end": 336
                    }
                  },
                  "loc": {
                    "start": 255,
                    "end": 336
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 341,
                      "end": 343
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 341,
                    "end": 343
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 348,
                      "end": 358
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 348,
                    "end": 358
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 363,
                      "end": 373
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 363,
                    "end": 373
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 378,
                      "end": 394
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 378,
                    "end": 394
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 399,
                      "end": 407
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 399,
                    "end": 407
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 412,
                      "end": 421
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 412,
                    "end": 421
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 426,
                      "end": 438
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 426,
                    "end": 438
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 443,
                      "end": 459
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 443,
                    "end": 459
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 464,
                      "end": 474
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 464,
                    "end": 474
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 479,
                      "end": 491
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 479,
                    "end": 491
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 496,
                      "end": 508
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 496,
                    "end": 508
                  }
                }
              ],
              "loc": {
                "start": 249,
                "end": 510
              }
            },
            "loc": {
              "start": 240,
              "end": 510
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 511,
                "end": 513
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 511,
              "end": 513
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 514,
                "end": 524
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 514,
              "end": 524
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 525,
                "end": 535
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 525,
              "end": 535
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 536,
                "end": 545
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 536,
              "end": 545
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 546,
                "end": 557
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 546,
              "end": 557
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 558,
                "end": 564
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 574,
                      "end": 584
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 571,
                    "end": 584
                  }
                }
              ],
              "loc": {
                "start": 565,
                "end": 586
              }
            },
            "loc": {
              "start": 558,
              "end": 586
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 587,
                "end": 592
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 606,
                        "end": 610
                      }
                    },
                    "loc": {
                      "start": 606,
                      "end": 610
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 624,
                            "end": 632
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 621,
                          "end": 632
                        }
                      }
                    ],
                    "loc": {
                      "start": 611,
                      "end": 638
                    }
                  },
                  "loc": {
                    "start": 599,
                    "end": 638
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 650,
                        "end": 654
                      }
                    },
                    "loc": {
                      "start": 650,
                      "end": 654
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 668,
                            "end": 676
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 665,
                          "end": 676
                        }
                      }
                    ],
                    "loc": {
                      "start": 655,
                      "end": 682
                    }
                  },
                  "loc": {
                    "start": 643,
                    "end": 682
                  }
                }
              ],
              "loc": {
                "start": 593,
                "end": 684
              }
            },
            "loc": {
              "start": 587,
              "end": 684
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 685,
                "end": 696
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 685,
              "end": 696
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 697,
                "end": 711
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 697,
              "end": 711
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 712,
                "end": 717
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 712,
              "end": 717
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 718,
                "end": 727
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 718,
              "end": 727
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 728,
                "end": 732
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 742,
                      "end": 750
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 739,
                    "end": 750
                  }
                }
              ],
              "loc": {
                "start": 733,
                "end": 752
              }
            },
            "loc": {
              "start": 728,
              "end": 752
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 753,
                "end": 767
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 753,
              "end": 767
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 768,
                "end": 773
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 768,
              "end": 773
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 774,
                "end": 777
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 784,
                      "end": 793
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 784,
                    "end": 793
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 798,
                      "end": 809
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 798,
                    "end": 809
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 814,
                      "end": 825
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 814,
                    "end": 825
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 830,
                      "end": 839
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 830,
                    "end": 839
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 844,
                      "end": 851
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 844,
                    "end": 851
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 856,
                      "end": 864
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 856,
                    "end": 864
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 869,
                      "end": 881
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 869,
                    "end": 881
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 886,
                      "end": 894
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 886,
                    "end": 894
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 899,
                      "end": 907
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 899,
                    "end": 907
                  }
                }
              ],
              "loc": {
                "start": 778,
                "end": 909
              }
            },
            "loc": {
              "start": 774,
              "end": 909
            }
          }
        ],
        "loc": {
          "start": 238,
          "end": 911
        }
      },
      "loc": {
        "start": 205,
        "end": 911
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 921,
          "end": 929
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 933,
            "end": 936
          }
        },
        "loc": {
          "start": 933,
          "end": 936
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 939,
                "end": 941
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 939,
              "end": 941
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 942,
                "end": 952
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 942,
              "end": 952
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 953,
                "end": 956
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 953,
              "end": 956
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 957,
                "end": 966
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 957,
              "end": 966
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 967,
                "end": 979
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 986,
                      "end": 988
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 986,
                    "end": 988
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 993,
                      "end": 1001
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 993,
                    "end": 1001
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1006,
                      "end": 1017
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1006,
                    "end": 1017
                  }
                }
              ],
              "loc": {
                "start": 980,
                "end": 1019
              }
            },
            "loc": {
              "start": 967,
              "end": 1019
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1020,
                "end": 1023
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 1030,
                      "end": 1035
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1030,
                    "end": 1035
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1040,
                      "end": 1052
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1040,
                    "end": 1052
                  }
                }
              ],
              "loc": {
                "start": 1024,
                "end": 1054
              }
            },
            "loc": {
              "start": 1020,
              "end": 1054
            }
          }
        ],
        "loc": {
          "start": 937,
          "end": 1056
        }
      },
      "loc": {
        "start": 912,
        "end": 1056
      }
    },
    "Team_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_list",
        "loc": {
          "start": 1066,
          "end": 1075
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 1079,
            "end": 1083
          }
        },
        "loc": {
          "start": 1079,
          "end": 1083
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1086,
                "end": 1088
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1086,
              "end": 1088
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 1089,
                "end": 1100
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1089,
              "end": 1100
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1101,
                "end": 1107
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1101,
              "end": 1107
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1108,
                "end": 1118
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1108,
              "end": 1118
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1119,
                "end": 1129
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1119,
              "end": 1129
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 1130,
                "end": 1148
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1130,
              "end": 1148
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1149,
                "end": 1158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1149,
              "end": 1158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 1159,
                "end": 1172
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1159,
              "end": 1172
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 1173,
                "end": 1185
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1173,
              "end": 1185
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 1186,
                "end": 1198
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1186,
              "end": 1198
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1199,
                "end": 1211
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1199,
              "end": 1211
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1212,
                "end": 1221
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1212,
              "end": 1221
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1222,
                "end": 1226
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 1236,
                      "end": 1244
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1233,
                    "end": 1244
                  }
                }
              ],
              "loc": {
                "start": 1227,
                "end": 1246
              }
            },
            "loc": {
              "start": 1222,
              "end": 1246
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1247,
                "end": 1259
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1266,
                      "end": 1268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1266,
                    "end": 1268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1273,
                      "end": 1281
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1273,
                    "end": 1281
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 1286,
                      "end": 1289
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1286,
                    "end": 1289
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1294,
                      "end": 1298
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1294,
                    "end": 1298
                  }
                }
              ],
              "loc": {
                "start": 1260,
                "end": 1300
              }
            },
            "loc": {
              "start": 1247,
              "end": 1300
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1301,
                "end": 1304
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 1311,
                      "end": 1324
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1311,
                    "end": 1324
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1329,
                      "end": 1338
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1329,
                    "end": 1338
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1343,
                      "end": 1354
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1343,
                    "end": 1354
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1359,
                      "end": 1368
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1359,
                    "end": 1368
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1373,
                      "end": 1382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1373,
                    "end": 1382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1387,
                      "end": 1394
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1387,
                    "end": 1394
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1399,
                      "end": 1411
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1399,
                    "end": 1411
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1416,
                      "end": 1424
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1416,
                    "end": 1424
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 1429,
                      "end": 1443
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1454,
                            "end": 1456
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1454,
                          "end": 1456
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 1465,
                            "end": 1475
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1465,
                          "end": 1475
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 1484,
                            "end": 1494
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1484,
                          "end": 1494
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 1503,
                            "end": 1510
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1503,
                          "end": 1510
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 1519,
                            "end": 1530
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1519,
                          "end": 1530
                        }
                      }
                    ],
                    "loc": {
                      "start": 1444,
                      "end": 1536
                    }
                  },
                  "loc": {
                    "start": 1429,
                    "end": 1536
                  }
                }
              ],
              "loc": {
                "start": 1305,
                "end": 1538
              }
            },
            "loc": {
              "start": 1301,
              "end": 1538
            }
          }
        ],
        "loc": {
          "start": 1084,
          "end": 1540
        }
      },
      "loc": {
        "start": 1057,
        "end": 1540
      }
    },
    "Team_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_nav",
        "loc": {
          "start": 1550,
          "end": 1558
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 1562,
            "end": 1566
          }
        },
        "loc": {
          "start": 1562,
          "end": 1566
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1569,
                "end": 1571
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1569,
              "end": 1571
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 1572,
                "end": 1583
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1572,
              "end": 1583
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1584,
                "end": 1590
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1584,
              "end": 1590
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 1591,
                "end": 1603
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1591,
              "end": 1603
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1604,
                "end": 1607
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 1614,
                      "end": 1627
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1614,
                    "end": 1627
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1632,
                      "end": 1641
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1632,
                    "end": 1641
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1646,
                      "end": 1657
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1646,
                    "end": 1657
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1662,
                      "end": 1671
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1662,
                    "end": 1671
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1676,
                      "end": 1685
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1676,
                    "end": 1685
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1690,
                      "end": 1697
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1690,
                    "end": 1697
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1702,
                      "end": 1714
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1702,
                    "end": 1714
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1719,
                      "end": 1727
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1719,
                    "end": 1727
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 1732,
                      "end": 1746
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1757,
                            "end": 1759
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1757,
                          "end": 1759
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 1768,
                            "end": 1778
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1768,
                          "end": 1778
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 1787,
                            "end": 1797
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1787,
                          "end": 1797
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 1806,
                            "end": 1813
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1806,
                          "end": 1813
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 1822,
                            "end": 1833
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1822,
                          "end": 1833
                        }
                      }
                    ],
                    "loc": {
                      "start": 1747,
                      "end": 1839
                    }
                  },
                  "loc": {
                    "start": 1732,
                    "end": 1839
                  }
                }
              ],
              "loc": {
                "start": 1608,
                "end": 1841
              }
            },
            "loc": {
              "start": 1604,
              "end": 1841
            }
          }
        ],
        "loc": {
          "start": 1567,
          "end": 1843
        }
      },
      "loc": {
        "start": 1541,
        "end": 1843
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 1853,
          "end": 1861
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 1865,
            "end": 1869
          }
        },
        "loc": {
          "start": 1865,
          "end": 1869
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1872,
                "end": 1874
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1872,
              "end": 1874
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1875,
                "end": 1885
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1875,
              "end": 1885
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1886,
                "end": 1896
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1886,
              "end": 1896
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 1897,
                "end": 1908
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1897,
              "end": 1908
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1909,
                "end": 1915
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1909,
              "end": 1915
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 1916,
                "end": 1921
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1916,
              "end": 1921
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 1922,
                "end": 1942
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1922,
              "end": 1942
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1943,
                "end": 1947
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1943,
              "end": 1947
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 1948,
                "end": 1960
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1948,
              "end": 1960
            }
          }
        ],
        "loc": {
          "start": 1870,
          "end": 1962
        }
      },
      "loc": {
        "start": 1844,
        "end": 1962
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "projectOrTeams",
      "loc": {
        "start": 1970,
        "end": 1984
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 1986,
              "end": 1991
            }
          },
          "loc": {
            "start": 1985,
            "end": 1991
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "ProjectOrTeamSearchInput",
              "loc": {
                "start": 1993,
                "end": 2017
              }
            },
            "loc": {
              "start": 1993,
              "end": 2017
            }
          },
          "loc": {
            "start": 1993,
            "end": 2018
          }
        },
        "directives": [],
        "loc": {
          "start": 1985,
          "end": 2018
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "projectOrTeams",
            "loc": {
              "start": 2024,
              "end": 2038
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 2039,
                  "end": 2044
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 2047,
                    "end": 2052
                  }
                },
                "loc": {
                  "start": 2046,
                  "end": 2052
                }
              },
              "loc": {
                "start": 2039,
                "end": 2052
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "edges",
                  "loc": {
                    "start": 2060,
                    "end": 2065
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "cursor",
                        "loc": {
                          "start": 2076,
                          "end": 2082
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2076,
                        "end": 2082
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 2091,
                          "end": 2095
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Project",
                                "loc": {
                                  "start": 2117,
                                  "end": 2124
                                }
                              },
                              "loc": {
                                "start": 2117,
                                "end": 2124
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Project_list",
                                    "loc": {
                                      "start": 2146,
                                      "end": 2158
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2143,
                                    "end": 2158
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2125,
                                "end": 2172
                              }
                            },
                            "loc": {
                              "start": 2110,
                              "end": 2172
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Team",
                                "loc": {
                                  "start": 2192,
                                  "end": 2196
                                }
                              },
                              "loc": {
                                "start": 2192,
                                "end": 2196
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Team_list",
                                    "loc": {
                                      "start": 2218,
                                      "end": 2227
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2215,
                                    "end": 2227
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2197,
                                "end": 2241
                              }
                            },
                            "loc": {
                              "start": 2185,
                              "end": 2241
                            }
                          }
                        ],
                        "loc": {
                          "start": 2096,
                          "end": 2251
                        }
                      },
                      "loc": {
                        "start": 2091,
                        "end": 2251
                      }
                    }
                  ],
                  "loc": {
                    "start": 2066,
                    "end": 2257
                  }
                },
                "loc": {
                  "start": 2060,
                  "end": 2257
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 2262,
                    "end": 2270
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 2281,
                          "end": 2292
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2281,
                        "end": 2292
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 2301,
                          "end": 2317
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2301,
                        "end": 2317
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorTeam",
                        "loc": {
                          "start": 2326,
                          "end": 2339
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2326,
                        "end": 2339
                      }
                    }
                  ],
                  "loc": {
                    "start": 2271,
                    "end": 2345
                  }
                },
                "loc": {
                  "start": 2262,
                  "end": 2345
                }
              }
            ],
            "loc": {
              "start": 2054,
              "end": 2349
            }
          },
          "loc": {
            "start": 2024,
            "end": 2349
          }
        }
      ],
      "loc": {
        "start": 2020,
        "end": 2351
      }
    },
    "loc": {
      "start": 1964,
      "end": 2351
    }
  },
  "variableValues": {},
  "path": {
    "key": "projectOrTeam_findMany"
  }
} as const;
