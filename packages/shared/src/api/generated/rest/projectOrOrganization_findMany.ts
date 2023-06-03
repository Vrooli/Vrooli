export const projectOrOrganization_findMany = {
  "fieldName": "projectOrOrganizations",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectOrOrganizations",
        "loc": {
          "start": 1986,
          "end": 2008
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2009,
              "end": 2014
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 2017,
                "end": 2022
              }
            },
            "loc": {
              "start": 2016,
              "end": 2022
            }
          },
          "loc": {
            "start": 2009,
            "end": 2022
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
                "start": 2030,
                "end": 2035
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
                      "start": 2046,
                      "end": 2052
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2046,
                    "end": 2052
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 2061,
                      "end": 2065
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
                              "start": 2087,
                              "end": 2094
                            }
                          },
                          "loc": {
                            "start": 2087,
                            "end": 2094
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
                                  "start": 2116,
                                  "end": 2128
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2113,
                                "end": 2128
                              }
                            }
                          ],
                          "loc": {
                            "start": 2095,
                            "end": 2142
                          }
                        },
                        "loc": {
                          "start": 2080,
                          "end": 2142
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Organization",
                            "loc": {
                              "start": 2162,
                              "end": 2174
                            }
                          },
                          "loc": {
                            "start": 2162,
                            "end": 2174
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
                                "value": "Organization_list",
                                "loc": {
                                  "start": 2196,
                                  "end": 2213
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2193,
                                "end": 2213
                              }
                            }
                          ],
                          "loc": {
                            "start": 2175,
                            "end": 2227
                          }
                        },
                        "loc": {
                          "start": 2155,
                          "end": 2227
                        }
                      }
                    ],
                    "loc": {
                      "start": 2066,
                      "end": 2237
                    }
                  },
                  "loc": {
                    "start": 2061,
                    "end": 2237
                  }
                }
              ],
              "loc": {
                "start": 2036,
                "end": 2243
              }
            },
            "loc": {
              "start": 2030,
              "end": 2243
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 2248,
                "end": 2256
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
                      "start": 2267,
                      "end": 2278
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2267,
                    "end": 2278
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 2287,
                      "end": 2303
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2287,
                    "end": 2303
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorOrganization",
                    "loc": {
                      "start": 2312,
                      "end": 2333
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2312,
                    "end": 2333
                  }
                }
              ],
              "loc": {
                "start": 2257,
                "end": 2339
              }
            },
            "loc": {
              "start": 2248,
              "end": 2339
            }
          }
        ],
        "loc": {
          "start": 2024,
          "end": 2343
        }
      },
      "loc": {
        "start": 1986,
        "end": 2343
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
                "value": "Organization",
                "loc": {
                  "start": 88,
                  "end": 100
                }
              },
              "loc": {
                "start": 88,
                "end": 100
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 114,
                      "end": 130
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 111,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 101,
                "end": 136
              }
            },
            "loc": {
              "start": 81,
              "end": 136
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
                  "start": 148,
                  "end": 152
                }
              },
              "loc": {
                "start": 148,
                "end": 152
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
                      "start": 166,
                      "end": 174
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 174
                  }
                }
              ],
              "loc": {
                "start": 153,
                "end": 180
              }
            },
            "loc": {
              "start": 141,
              "end": 180
            }
          }
        ],
        "loc": {
          "start": 75,
          "end": 182
        }
      },
      "loc": {
        "start": 69,
        "end": 182
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 183,
          "end": 186
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
                "start": 193,
                "end": 202
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 193,
              "end": 202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 207,
                "end": 216
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 207,
              "end": 216
            }
          }
        ],
        "loc": {
          "start": 187,
          "end": 218
        }
      },
      "loc": {
        "start": 183,
        "end": 218
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 266,
          "end": 268
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 266,
        "end": 268
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 269,
          "end": 275
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 269,
        "end": 275
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 276,
          "end": 286
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 276,
        "end": 286
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 287,
          "end": 297
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 287,
        "end": 297
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isOpenToNewMembers",
        "loc": {
          "start": 298,
          "end": 316
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 298,
        "end": 316
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 317,
          "end": 326
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 317,
        "end": 326
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 327,
          "end": 340
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 327,
        "end": 340
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "membersCount",
        "loc": {
          "start": 341,
          "end": 353
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 341,
        "end": 353
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 354,
          "end": 366
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 354,
        "end": 366
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 367,
          "end": 376
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 367,
        "end": 376
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 377,
          "end": 381
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
                "start": 391,
                "end": 399
              }
            },
            "directives": [],
            "loc": {
              "start": 388,
              "end": 399
            }
          }
        ],
        "loc": {
          "start": 382,
          "end": 401
        }
      },
      "loc": {
        "start": 377,
        "end": 401
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 402,
          "end": 414
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
                "start": 421,
                "end": 423
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 421,
              "end": 423
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 428,
                "end": 436
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 428,
              "end": 436
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 441,
                "end": 444
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 441,
              "end": 444
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 449,
                "end": 453
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 449,
              "end": 453
            }
          }
        ],
        "loc": {
          "start": 415,
          "end": 455
        }
      },
      "loc": {
        "start": 402,
        "end": 455
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 456,
          "end": 459
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
                "start": 466,
                "end": 479
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 466,
              "end": 479
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 484,
                "end": 493
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 484,
              "end": 493
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 498,
                "end": 509
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 498,
              "end": 509
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 514,
                "end": 523
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 514,
              "end": 523
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 528,
                "end": 537
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 528,
              "end": 537
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 542,
                "end": 549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 542,
              "end": 549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 554,
                "end": 566
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 554,
              "end": 566
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 571,
                "end": 579
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 571,
              "end": 579
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 584,
                "end": 598
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
                      "start": 609,
                      "end": 611
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 609,
                    "end": 611
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 620,
                      "end": 630
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 620,
                    "end": 630
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 639,
                      "end": 649
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 639,
                    "end": 649
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 658,
                      "end": 665
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 658,
                    "end": 665
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 674,
                      "end": 685
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 674,
                    "end": 685
                  }
                }
              ],
              "loc": {
                "start": 599,
                "end": 691
              }
            },
            "loc": {
              "start": 584,
              "end": 691
            }
          }
        ],
        "loc": {
          "start": 460,
          "end": 693
        }
      },
      "loc": {
        "start": 456,
        "end": 693
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 740,
          "end": 742
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 740,
        "end": 742
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 743,
          "end": 749
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 743,
        "end": 749
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 750,
          "end": 753
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
                "start": 760,
                "end": 773
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 760,
              "end": 773
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 778,
                "end": 787
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 778,
              "end": 787
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 792,
                "end": 803
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 792,
              "end": 803
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 808,
                "end": 817
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 808,
              "end": 817
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 822,
                "end": 831
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 822,
              "end": 831
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 836,
                "end": 843
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 836,
              "end": 843
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 848,
                "end": 860
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 848,
              "end": 860
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 865,
                "end": 873
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 865,
              "end": 873
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 878,
                "end": 892
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
                      "start": 903,
                      "end": 905
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 903,
                    "end": 905
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 914,
                      "end": 924
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 914,
                    "end": 924
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 933,
                      "end": 943
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 933,
                    "end": 943
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 952,
                      "end": 959
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 952,
                    "end": 959
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 968,
                      "end": 979
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 968,
                    "end": 979
                  }
                }
              ],
              "loc": {
                "start": 893,
                "end": 985
              }
            },
            "loc": {
              "start": 878,
              "end": 985
            }
          }
        ],
        "loc": {
          "start": 754,
          "end": 987
        }
      },
      "loc": {
        "start": 750,
        "end": 987
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 1025,
          "end": 1033
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
                "start": 1040,
                "end": 1052
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
                      "start": 1063,
                      "end": 1065
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1063,
                    "end": 1065
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1074,
                      "end": 1082
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1074,
                    "end": 1082
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1091,
                      "end": 1102
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1091,
                    "end": 1102
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1111,
                      "end": 1115
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1111,
                    "end": 1115
                  }
                }
              ],
              "loc": {
                "start": 1053,
                "end": 1121
              }
            },
            "loc": {
              "start": 1040,
              "end": 1121
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1126,
                "end": 1128
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1126,
              "end": 1128
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1133,
                "end": 1143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1133,
              "end": 1143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1148,
                "end": 1158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1148,
              "end": 1158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 1163,
                "end": 1179
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1163,
              "end": 1179
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1184,
                "end": 1192
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1184,
              "end": 1192
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1197,
                "end": 1206
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1197,
              "end": 1206
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1211,
                "end": 1223
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1211,
              "end": 1223
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 1228,
                "end": 1244
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1228,
              "end": 1244
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 1249,
                "end": 1259
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1249,
              "end": 1259
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1264,
                "end": 1276
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1264,
              "end": 1276
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1281,
                "end": 1293
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1281,
              "end": 1293
            }
          }
        ],
        "loc": {
          "start": 1034,
          "end": 1295
        }
      },
      "loc": {
        "start": 1025,
        "end": 1295
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1296,
          "end": 1298
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1296,
        "end": 1298
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1299,
          "end": 1309
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1299,
        "end": 1309
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1310,
          "end": 1320
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1310,
        "end": 1320
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1321,
          "end": 1330
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1321,
        "end": 1330
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1331,
          "end": 1342
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1331,
        "end": 1342
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1343,
          "end": 1349
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
                "start": 1359,
                "end": 1369
              }
            },
            "directives": [],
            "loc": {
              "start": 1356,
              "end": 1369
            }
          }
        ],
        "loc": {
          "start": 1350,
          "end": 1371
        }
      },
      "loc": {
        "start": 1343,
        "end": 1371
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1372,
          "end": 1377
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
                "value": "Organization",
                "loc": {
                  "start": 1391,
                  "end": 1403
                }
              },
              "loc": {
                "start": 1391,
                "end": 1403
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1417,
                      "end": 1433
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1414,
                    "end": 1433
                  }
                }
              ],
              "loc": {
                "start": 1404,
                "end": 1439
              }
            },
            "loc": {
              "start": 1384,
              "end": 1439
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
                  "start": 1451,
                  "end": 1455
                }
              },
              "loc": {
                "start": 1451,
                "end": 1455
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
                      "start": 1469,
                      "end": 1477
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1466,
                    "end": 1477
                  }
                }
              ],
              "loc": {
                "start": 1456,
                "end": 1483
              }
            },
            "loc": {
              "start": 1444,
              "end": 1483
            }
          }
        ],
        "loc": {
          "start": 1378,
          "end": 1485
        }
      },
      "loc": {
        "start": 1372,
        "end": 1485
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1486,
          "end": 1497
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1486,
        "end": 1497
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1498,
          "end": 1512
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1498,
        "end": 1512
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1513,
          "end": 1518
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1513,
        "end": 1518
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1519,
          "end": 1528
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1519,
        "end": 1528
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1529,
          "end": 1533
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
                "start": 1543,
                "end": 1551
              }
            },
            "directives": [],
            "loc": {
              "start": 1540,
              "end": 1551
            }
          }
        ],
        "loc": {
          "start": 1534,
          "end": 1553
        }
      },
      "loc": {
        "start": 1529,
        "end": 1553
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1554,
          "end": 1568
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1554,
        "end": 1568
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1569,
          "end": 1574
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1569,
        "end": 1574
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1575,
          "end": 1578
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
                "start": 1585,
                "end": 1594
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1585,
              "end": 1594
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1599,
                "end": 1610
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1599,
              "end": 1610
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 1615,
                "end": 1626
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1615,
              "end": 1626
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1631,
                "end": 1640
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1631,
              "end": 1640
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1645,
                "end": 1652
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1645,
              "end": 1652
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1657,
                "end": 1665
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1657,
              "end": 1665
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1670,
                "end": 1682
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1670,
              "end": 1682
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1687,
                "end": 1695
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1687,
              "end": 1695
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1700,
                "end": 1708
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1700,
              "end": 1708
            }
          }
        ],
        "loc": {
          "start": 1579,
          "end": 1710
        }
      },
      "loc": {
        "start": 1575,
        "end": 1710
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1740,
          "end": 1742
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1740,
        "end": 1742
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1743,
          "end": 1753
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1743,
        "end": 1753
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 1754,
          "end": 1757
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1754,
        "end": 1757
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1758,
          "end": 1767
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1758,
        "end": 1767
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 1768,
          "end": 1780
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
                "start": 1787,
                "end": 1789
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1787,
              "end": 1789
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 1794,
                "end": 1802
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1794,
              "end": 1802
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1807,
                "end": 1818
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1807,
              "end": 1818
            }
          }
        ],
        "loc": {
          "start": 1781,
          "end": 1820
        }
      },
      "loc": {
        "start": 1768,
        "end": 1820
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1821,
          "end": 1824
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
                "start": 1831,
                "end": 1836
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1831,
              "end": 1836
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1841,
                "end": 1853
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1841,
              "end": 1853
            }
          }
        ],
        "loc": {
          "start": 1825,
          "end": 1855
        }
      },
      "loc": {
        "start": 1821,
        "end": 1855
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1886,
          "end": 1888
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1886,
        "end": 1888
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 1889,
          "end": 1894
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1889,
        "end": 1894
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 1895,
          "end": 1899
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1895,
        "end": 1899
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1900,
          "end": 1906
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1900,
        "end": 1906
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
                      "value": "Organization",
                      "loc": {
                        "start": 88,
                        "end": 100
                      }
                    },
                    "loc": {
                      "start": 88,
                      "end": 100
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 114,
                            "end": 130
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 111,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 101,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 81,
                    "end": 136
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
                        "start": 148,
                        "end": 152
                      }
                    },
                    "loc": {
                      "start": 148,
                      "end": 152
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
                            "start": 166,
                            "end": 174
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 163,
                          "end": 174
                        }
                      }
                    ],
                    "loc": {
                      "start": 153,
                      "end": 180
                    }
                  },
                  "loc": {
                    "start": 141,
                    "end": 180
                  }
                }
              ],
              "loc": {
                "start": 75,
                "end": 182
              }
            },
            "loc": {
              "start": 69,
              "end": 182
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 183,
                "end": 186
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
                      "start": 193,
                      "end": 202
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 193,
                    "end": 202
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 207,
                      "end": 216
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 207,
                    "end": 216
                  }
                }
              ],
              "loc": {
                "start": 187,
                "end": 218
              }
            },
            "loc": {
              "start": 183,
              "end": 218
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 220
        }
      },
      "loc": {
        "start": 1,
        "end": 220
      }
    },
    "Organization_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_list",
        "loc": {
          "start": 230,
          "end": 247
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 251,
            "end": 263
          }
        },
        "loc": {
          "start": 251,
          "end": 263
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
                "start": 266,
                "end": 268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 266,
              "end": 268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 269,
                "end": 275
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 269,
              "end": 275
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 276,
                "end": 286
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 276,
              "end": 286
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 287,
                "end": 297
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 287,
              "end": 297
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 298,
                "end": 316
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 298,
              "end": 316
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 317,
                "end": 326
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 317,
              "end": 326
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 327,
                "end": 340
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 327,
              "end": 340
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 341,
                "end": 353
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 341,
              "end": 353
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 354,
                "end": 366
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 354,
              "end": 366
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 367,
                "end": 376
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 367,
              "end": 376
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 377,
                "end": 381
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
                      "start": 391,
                      "end": 399
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 388,
                    "end": 399
                  }
                }
              ],
              "loc": {
                "start": 382,
                "end": 401
              }
            },
            "loc": {
              "start": 377,
              "end": 401
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 402,
                "end": 414
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
                      "start": 421,
                      "end": 423
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 421,
                    "end": 423
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 428,
                      "end": 436
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 428,
                    "end": 436
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 441,
                      "end": 444
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 441,
                    "end": 444
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 449,
                      "end": 453
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 449,
                    "end": 453
                  }
                }
              ],
              "loc": {
                "start": 415,
                "end": 455
              }
            },
            "loc": {
              "start": 402,
              "end": 455
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 456,
                "end": 459
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
                      "start": 466,
                      "end": 479
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 466,
                    "end": 479
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 484,
                      "end": 493
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 484,
                    "end": 493
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 498,
                      "end": 509
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 498,
                    "end": 509
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 514,
                      "end": 523
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 514,
                    "end": 523
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 528,
                      "end": 537
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 528,
                    "end": 537
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 542,
                      "end": 549
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 542,
                    "end": 549
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 554,
                      "end": 566
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 554,
                    "end": 566
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 571,
                      "end": 579
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 571,
                    "end": 579
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 584,
                      "end": 598
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
                            "start": 609,
                            "end": 611
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 609,
                          "end": 611
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 620,
                            "end": 630
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 620,
                          "end": 630
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 639,
                            "end": 649
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 639,
                          "end": 649
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 658,
                            "end": 665
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 658,
                          "end": 665
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 674,
                            "end": 685
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 674,
                          "end": 685
                        }
                      }
                    ],
                    "loc": {
                      "start": 599,
                      "end": 691
                    }
                  },
                  "loc": {
                    "start": 584,
                    "end": 691
                  }
                }
              ],
              "loc": {
                "start": 460,
                "end": 693
              }
            },
            "loc": {
              "start": 456,
              "end": 693
            }
          }
        ],
        "loc": {
          "start": 264,
          "end": 695
        }
      },
      "loc": {
        "start": 221,
        "end": 695
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 705,
          "end": 721
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 725,
            "end": 737
          }
        },
        "loc": {
          "start": 725,
          "end": 737
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
                "start": 740,
                "end": 742
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 740,
              "end": 742
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 743,
                "end": 749
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 743,
              "end": 749
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 750,
                "end": 753
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
                      "start": 760,
                      "end": 773
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 760,
                    "end": 773
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 778,
                      "end": 787
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 778,
                    "end": 787
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 792,
                      "end": 803
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 792,
                    "end": 803
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 808,
                      "end": 817
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 808,
                    "end": 817
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 822,
                      "end": 831
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 822,
                    "end": 831
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 836,
                      "end": 843
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 836,
                    "end": 843
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 848,
                      "end": 860
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 848,
                    "end": 860
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 865,
                      "end": 873
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 865,
                    "end": 873
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 878,
                      "end": 892
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
                            "start": 903,
                            "end": 905
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 903,
                          "end": 905
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 914,
                            "end": 924
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 914,
                          "end": 924
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 933,
                            "end": 943
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 933,
                          "end": 943
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 952,
                            "end": 959
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 952,
                          "end": 959
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 968,
                            "end": 979
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 968,
                          "end": 979
                        }
                      }
                    ],
                    "loc": {
                      "start": 893,
                      "end": 985
                    }
                  },
                  "loc": {
                    "start": 878,
                    "end": 985
                  }
                }
              ],
              "loc": {
                "start": 754,
                "end": 987
              }
            },
            "loc": {
              "start": 750,
              "end": 987
            }
          }
        ],
        "loc": {
          "start": 738,
          "end": 989
        }
      },
      "loc": {
        "start": 696,
        "end": 989
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 999,
          "end": 1011
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 1015,
            "end": 1022
          }
        },
        "loc": {
          "start": 1015,
          "end": 1022
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
                "start": 1025,
                "end": 1033
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
                      "start": 1040,
                      "end": 1052
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
                            "start": 1063,
                            "end": 1065
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1063,
                          "end": 1065
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1074,
                            "end": 1082
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1074,
                          "end": 1082
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1091,
                            "end": 1102
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1091,
                          "end": 1102
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1111,
                            "end": 1115
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1111,
                          "end": 1115
                        }
                      }
                    ],
                    "loc": {
                      "start": 1053,
                      "end": 1121
                    }
                  },
                  "loc": {
                    "start": 1040,
                    "end": 1121
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1126,
                      "end": 1128
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1126,
                    "end": 1128
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1133,
                      "end": 1143
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1133,
                    "end": 1143
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1148,
                      "end": 1158
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1148,
                    "end": 1158
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 1163,
                      "end": 1179
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1163,
                    "end": 1179
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1184,
                      "end": 1192
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1184,
                    "end": 1192
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1197,
                      "end": 1206
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1197,
                    "end": 1206
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1211,
                      "end": 1223
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1211,
                    "end": 1223
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 1228,
                      "end": 1244
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1228,
                    "end": 1244
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 1249,
                      "end": 1259
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1249,
                    "end": 1259
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1264,
                      "end": 1276
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1264,
                    "end": 1276
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1281,
                      "end": 1293
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1281,
                    "end": 1293
                  }
                }
              ],
              "loc": {
                "start": 1034,
                "end": 1295
              }
            },
            "loc": {
              "start": 1025,
              "end": 1295
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1296,
                "end": 1298
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1296,
              "end": 1298
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1299,
                "end": 1309
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1299,
              "end": 1309
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1310,
                "end": 1320
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1310,
              "end": 1320
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1321,
                "end": 1330
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1321,
              "end": 1330
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1331,
                "end": 1342
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1331,
              "end": 1342
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1343,
                "end": 1349
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
                      "start": 1359,
                      "end": 1369
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1356,
                    "end": 1369
                  }
                }
              ],
              "loc": {
                "start": 1350,
                "end": 1371
              }
            },
            "loc": {
              "start": 1343,
              "end": 1371
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1372,
                "end": 1377
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
                      "value": "Organization",
                      "loc": {
                        "start": 1391,
                        "end": 1403
                      }
                    },
                    "loc": {
                      "start": 1391,
                      "end": 1403
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1417,
                            "end": 1433
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1414,
                          "end": 1433
                        }
                      }
                    ],
                    "loc": {
                      "start": 1404,
                      "end": 1439
                    }
                  },
                  "loc": {
                    "start": 1384,
                    "end": 1439
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
                        "start": 1451,
                        "end": 1455
                      }
                    },
                    "loc": {
                      "start": 1451,
                      "end": 1455
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
                            "start": 1469,
                            "end": 1477
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1466,
                          "end": 1477
                        }
                      }
                    ],
                    "loc": {
                      "start": 1456,
                      "end": 1483
                    }
                  },
                  "loc": {
                    "start": 1444,
                    "end": 1483
                  }
                }
              ],
              "loc": {
                "start": 1378,
                "end": 1485
              }
            },
            "loc": {
              "start": 1372,
              "end": 1485
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1486,
                "end": 1497
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1486,
              "end": 1497
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1498,
                "end": 1512
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1498,
              "end": 1512
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1513,
                "end": 1518
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1513,
              "end": 1518
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1519,
                "end": 1528
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1519,
              "end": 1528
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1529,
                "end": 1533
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
                      "start": 1543,
                      "end": 1551
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1540,
                    "end": 1551
                  }
                }
              ],
              "loc": {
                "start": 1534,
                "end": 1553
              }
            },
            "loc": {
              "start": 1529,
              "end": 1553
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1554,
                "end": 1568
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1554,
              "end": 1568
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1569,
                "end": 1574
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1569,
              "end": 1574
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1575,
                "end": 1578
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
                      "start": 1585,
                      "end": 1594
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1585,
                    "end": 1594
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1599,
                      "end": 1610
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1599,
                    "end": 1610
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1615,
                      "end": 1626
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1615,
                    "end": 1626
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1631,
                      "end": 1640
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1631,
                    "end": 1640
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1645,
                      "end": 1652
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1645,
                    "end": 1652
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1657,
                      "end": 1665
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1657,
                    "end": 1665
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1670,
                      "end": 1682
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1670,
                    "end": 1682
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1687,
                      "end": 1695
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1687,
                    "end": 1695
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1700,
                      "end": 1708
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1700,
                    "end": 1708
                  }
                }
              ],
              "loc": {
                "start": 1579,
                "end": 1710
              }
            },
            "loc": {
              "start": 1575,
              "end": 1710
            }
          }
        ],
        "loc": {
          "start": 1023,
          "end": 1712
        }
      },
      "loc": {
        "start": 990,
        "end": 1712
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 1722,
          "end": 1730
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 1734,
            "end": 1737
          }
        },
        "loc": {
          "start": 1734,
          "end": 1737
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
                "start": 1740,
                "end": 1742
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1740,
              "end": 1742
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1743,
                "end": 1753
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1743,
              "end": 1753
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 1754,
                "end": 1757
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1754,
              "end": 1757
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1758,
                "end": 1767
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1758,
              "end": 1767
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1768,
                "end": 1780
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
                      "start": 1787,
                      "end": 1789
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1787,
                    "end": 1789
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1794,
                      "end": 1802
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1794,
                    "end": 1802
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1807,
                      "end": 1818
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1807,
                    "end": 1818
                  }
                }
              ],
              "loc": {
                "start": 1781,
                "end": 1820
              }
            },
            "loc": {
              "start": 1768,
              "end": 1820
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1821,
                "end": 1824
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
                      "start": 1831,
                      "end": 1836
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1831,
                    "end": 1836
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1841,
                      "end": 1853
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1841,
                    "end": 1853
                  }
                }
              ],
              "loc": {
                "start": 1825,
                "end": 1855
              }
            },
            "loc": {
              "start": 1821,
              "end": 1855
            }
          }
        ],
        "loc": {
          "start": 1738,
          "end": 1857
        }
      },
      "loc": {
        "start": 1713,
        "end": 1857
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 1867,
          "end": 1875
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 1879,
            "end": 1883
          }
        },
        "loc": {
          "start": 1879,
          "end": 1883
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
                "start": 1886,
                "end": 1888
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1886,
              "end": 1888
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 1889,
                "end": 1894
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1889,
              "end": 1894
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1895,
                "end": 1899
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1895,
              "end": 1899
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1900,
                "end": 1906
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1900,
              "end": 1906
            }
          }
        ],
        "loc": {
          "start": 1884,
          "end": 1908
        }
      },
      "loc": {
        "start": 1858,
        "end": 1908
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "projectOrOrganizations",
      "loc": {
        "start": 1916,
        "end": 1938
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
              "start": 1940,
              "end": 1945
            }
          },
          "loc": {
            "start": 1939,
            "end": 1945
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "ProjectOrOrganizationSearchInput",
              "loc": {
                "start": 1947,
                "end": 1979
              }
            },
            "loc": {
              "start": 1947,
              "end": 1979
            }
          },
          "loc": {
            "start": 1947,
            "end": 1980
          }
        },
        "directives": [],
        "loc": {
          "start": 1939,
          "end": 1980
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
            "value": "projectOrOrganizations",
            "loc": {
              "start": 1986,
              "end": 2008
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 2009,
                  "end": 2014
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 2017,
                    "end": 2022
                  }
                },
                "loc": {
                  "start": 2016,
                  "end": 2022
                }
              },
              "loc": {
                "start": 2009,
                "end": 2022
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
                    "start": 2030,
                    "end": 2035
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
                          "start": 2046,
                          "end": 2052
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2046,
                        "end": 2052
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 2061,
                          "end": 2065
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
                                  "start": 2087,
                                  "end": 2094
                                }
                              },
                              "loc": {
                                "start": 2087,
                                "end": 2094
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
                                      "start": 2116,
                                      "end": 2128
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2113,
                                    "end": 2128
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2095,
                                "end": 2142
                              }
                            },
                            "loc": {
                              "start": 2080,
                              "end": 2142
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Organization",
                                "loc": {
                                  "start": 2162,
                                  "end": 2174
                                }
                              },
                              "loc": {
                                "start": 2162,
                                "end": 2174
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
                                    "value": "Organization_list",
                                    "loc": {
                                      "start": 2196,
                                      "end": 2213
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2193,
                                    "end": 2213
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2175,
                                "end": 2227
                              }
                            },
                            "loc": {
                              "start": 2155,
                              "end": 2227
                            }
                          }
                        ],
                        "loc": {
                          "start": 2066,
                          "end": 2237
                        }
                      },
                      "loc": {
                        "start": 2061,
                        "end": 2237
                      }
                    }
                  ],
                  "loc": {
                    "start": 2036,
                    "end": 2243
                  }
                },
                "loc": {
                  "start": 2030,
                  "end": 2243
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 2248,
                    "end": 2256
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
                          "start": 2267,
                          "end": 2278
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2267,
                        "end": 2278
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 2287,
                          "end": 2303
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2287,
                        "end": 2303
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorOrganization",
                        "loc": {
                          "start": 2312,
                          "end": 2333
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2312,
                        "end": 2333
                      }
                    }
                  ],
                  "loc": {
                    "start": 2257,
                    "end": 2339
                  }
                },
                "loc": {
                  "start": 2248,
                  "end": 2339
                }
              }
            ],
            "loc": {
              "start": 2024,
              "end": 2343
            }
          },
          "loc": {
            "start": 1986,
            "end": 2343
          }
        }
      ],
      "loc": {
        "start": 1982,
        "end": 2345
      }
    },
    "loc": {
      "start": 1910,
      "end": 2345
    }
  },
  "variableValues": {},
  "path": {
    "key": "projectOrOrganization_findMany"
  }
} as const;
